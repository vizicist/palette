package kit

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// paramValuesForConcurrencyTest installs a small set of global parameter
// definitions and returns a ParamValues populated from them.
func paramValuesForConcurrencyTest(t *testing.T) (*ParamValues, []string) {
	t.Helper()

	oldParamDefs := ParamDefs
	t.Cleanup(func() { ParamDefs = oldParamDefs })

	ParamDefs = map[string]ParamDef{}
	names := []string{}
	for i := 0; i < 24; i++ {
		name := fmt.Sprintf("global.concurrencytest%02d", i)
		ParamDefs[name] = ParamDef{
			TypedParamDef: ParamDefString{},
			Category:      "global",
			Init:          "init",
		}
		names = append(names, name)
	}

	vals := NewParamValues()
	for _, name := range names {
		if err := vals.SetParamWithString(name, "init"); err != nil {
			t.Fatalf("SetParamWithString %s: %v", name, err)
		}
	}
	return vals, names
}

// The parameter map is reached from several goroutines at once - every HTTP
// request runs on its own, NATS is a second entry point, and the scheduler
// loads presets on a third - so reads and writes genuinely overlap.
//
// Two separate bugs lived here and this test covers both:
//
//   - Save ranged the values map holding no lock at all while SetParamWithString
//     wrote to it under the write lock. That is "concurrent map iteration and
//     map write", a fatal runtime error rather than a panic, so no recover()
//     anywhere can keep the engine alive. Under -race it is reported; without
//     -race it eventually kills the test binary outright.
//
//   - GetWithPrefix took the read lock and then called ParamValueAsString,
//     which took it again. sync.RWMutex forbids that: once a writer's Lock is
//     pending, the nested RLock queues behind it and the goroutines wait on
//     each other forever. Here that shows up as the deadline expiring.
func TestParamValuesConcurrentReadersAndWriters(t *testing.T) {

	vals, names := paramValuesForConcurrencyTest(t)

	const rounds = 300
	done := make(chan struct{})

	go func() {
		defer close(done)
		var wg sync.WaitGroup

		for w := 0; w < 3; w++ {
			wg.Add(1)
			go func(w int) {
				defer wg.Done()
				for i := 0; i < rounds; i++ {
					name := names[i%len(names)]
					if err := vals.SetParamWithString(name, fmt.Sprintf("v%d-%d", w, i)); err != nil {
						t.Error(err)
						return
					}
				}
			}(w)
		}

		// One reader per method that walks the whole map.
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < rounds; i++ {
				if _, err := vals.GetWithPrefix("global."); err != nil {
					t.Error(err)
					return
				}
			}
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < rounds; i++ {
				// What Save snapshots before it marshals.
				if got := vals.paramsForCategory("global"); len(got) == 0 {
					t.Error("paramsForCategory returned nothing")
					return
				}
			}
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < rounds; i++ {
				_, _ = vals.ParamValueAsString(names[i%len(names)])
				_ = vals.ParamNames()
				_ = vals.Exists(names[i%len(names)])
			}
		}()

		wg.Wait()
	}()

	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("timed out: parameter access deadlocked, which is what a recursive RLock behind a pending writer does")
	}
}

// paramsForCategory must still select the right parameters, not just be
// thread-safe: Save writes whatever it returns.
func TestParamsForCategorySelectsCategory(t *testing.T) {

	vals, names := paramValuesForConcurrencyTest(t)

	// A parameter that must not appear in a "global" save.
	ParamDefs["sound.concurrencytest"] = ParamDef{
		TypedParamDef: ParamDefString{},
		Category:      "sound",
		Init:          "init",
	}
	if err := vals.SetParamWithString("sound.concurrencytest", "nope"); err != nil {
		t.Fatal(err)
	}
	// Names starting with _ are deliberately not saved.
	ParamDefs["global._hidden"] = ParamDef{
		TypedParamDef: ParamDefString{},
		Category:      "global",
		Init:          "init",
	}
	if err := vals.SetParamWithString("global._hidden", "secret"); err != nil {
		t.Fatal(err)
	}

	got := vals.paramsForCategory("global")

	if len(got) != len(names) {
		t.Fatalf("got %d global params, want %d", len(got), len(names))
	}
	if _, ok := got["sound.concurrencytest"]; ok {
		t.Error("a sound param leaked into a global save")
	}
	if _, ok := got["global._hidden"]; ok {
		t.Error("an _-prefixed param was saved")
	}
	for _, name := range names {
		if got[name] != "init" {
			t.Fatalf("%s = %q, want %q", name, got[name], "init")
		}
	}
}
