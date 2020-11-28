#include "PaletteAll.h"

// These get initialized in Scheduler constructor
int Scheduler::ClicksPerSecond;
int Scheduler::CurrentClick;
int Scheduler::CurrentClickOffset;
int Scheduler::CurrentMilliOffset;
bool Scheduler::Debug;

int Scheduler::CurrentMilli;

int GlobalPitchOffset = 0;
bool NoMultiNotes = true;
bool DoControllers = true;

bool DebugPlaying = false;

Scheduler::Scheduler(PaletteHost* p) {
	_paletteHost = p;
	m_running = false;
	Debug = false;

	InitializeClicksPerSecond(DEFAULT_CLICKS_PER_SECOND);

	NosuchLockInit(&_callback_mutex, "callback");
}

Scheduler::~Scheduler() {
	NosuchDebug("Scheduler destructor!");
}

void Scheduler::RunEveryMillisecondOrSo(Timestamp timestamp) {

	if (m_running == false) {
		return;
	}

	// We don't want to collect a whole bunch of blocked callbacks,
	// so if we can't get the lock, we just give up.
	int err = TryLockCallback();
	if (err != 0) {
		return;
	}

	AdvanceTime(timestamp);

	static int lastdump = 0;
	// NosuchDebug messages aren't flushed to the log right away, to avoid
	// screwed up timing continuously.  I.e. we only screw up the timing every 5 seconds
	if ( NosuchDebugAutoFlush==false && (timestamp - lastdump) > 5000 ) {
		lastdump = timestamp;
		NosuchDebugDumpLog();
	}

	UnlockCallback();

	return;
}

static char easytolower(char in){
  if(in<='Z' && in>='A')
    return in-('Z'-'z');
  return in;
} 

static std::string lowercase(std::string s) {
	std::string lc = s;
	std::transform(lc.begin(), lc.end(), lc.begin(), easytolower);
	return lc;
}

void Scheduler::Stop() {
	if ( m_running == true ) {
		m_running = false;
	}
}

static int
smoothTo(int tovalue, int fromvalue) {
	int dv = tovalue - fromvalue;
	if (dv == 0 || dv == 1 || dv == -1 ) {
		return tovalue;
	}
	return (tovalue + fromvalue) / 2;
}

std::string Scheduler::DebugString() {

	std::string s;
	s = "Scheduler (\n";
	s += "   }";
	return s;
}

////////////////////////////////////////////////////////////////////
// I think all the code that manipulates time values is here.
////////////////////////////////////////////////////////////////////

#define ClicksPerMillisecond (ClicksPerSecond / 1000.0)

void Scheduler::AdvanceTime(Timestamp timestamp) {
	CurrentMilli = timestamp;
	NosuchAssert(m_running == true);

	click_t sofar = getClicksForTimestamp(timestamp);
	if (sofar <= CurrentClick) {
		// Clicks haven't advanced
		return;
	}
	CurrentClick = sofar;
}

void Scheduler::InitializeClicksPerSecond(int clkpersec) {
	ClicksPerSecond = clkpersec;
	CurrentClick = (int)(0.5 + CurrentMilli * ClicksPerMillisecond);
	CurrentMilliOffset = 0;
	CurrentClickOffset = 0;
}

void Scheduler::ChangeClicksPerSecond(int clkpersec) {
	if (clkpersec < MIN_CLICKS_PER_SECOND) {
		clkpersec = MIN_CLICKS_PER_SECOND;
	}
	if (clkpersec > MAX_CLICKS_PER_SECOND) {
		clkpersec = MAX_CLICKS_PER_SECOND;
	}
	NosuchDebug("Changing ClicksPerSecond to %d", clkpersec);
	// NosuchDebug("CurrentMilliOffset was %d", CurrentMilliOffset);
	// NosuchDebug("CurrentClickOffset was %d", CurrentClickOffset);
	// NosuchDebug("CurrentMilli was %d", CurrentMilli);
	// NosuchDebug("CurrentClick was %d", CurrentClick);
	CurrentMilliOffset = CurrentMilli;
	CurrentClickOffset = CurrentClick;
	ClicksPerSecond = clkpersec;
	// NosuchDebug("");
	// NosuchDebug("CurrentMilliOffset is now %d", CurrentMilliOffset);
	// NosuchDebug("CurrentClickOffset is now %d", CurrentClickOffset);
	// NosuchDebug("CurrentMilli is now %d", CurrentMilli);
	// NosuchDebug("CurrentClick is now %d", CurrentClick);
}

click_t Scheduler::getClicksForTimestamp(Timestamp timestamp) {
	click_t sofar = CurrentClickOffset + (click_t)(0.5 + (timestamp-CurrentMilliOffset) * ClicksPerMillisecond);
	return sofar;
}