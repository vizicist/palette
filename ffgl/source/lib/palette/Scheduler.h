#ifndef _SCHEDULER_H
#define _SCHEDULER_H

#include "NosuchException.h"
#include <list>
#include <map>
#include <algorithm>

#define IN_QUEUE_SIZE 1024
#define OUT_QUEUE_SIZE 1024

#define NOSUCH_SID -1

#define DEFAULT_CLICKS_PER_SECOND 192
#define MIN_CLICKS_PER_SECOND (192/16)
#define MAX_CLICKS_PER_SECOND (192*16)

class Scheduler;
class TrackedCursor;
class Layer;
class Palette;

typedef int click_t;
typedef long Timestamp;

class Scheduler {
public:

	static int ClicksPerSecond;
	static int CurrentClick;
	static int CurrentMilli;
	static int CurrentMilliOffset;
	static int CurrentClickOffset;
	static bool Debug;

	static void InitializeClicksPerSecond(int clkpersec);
	static void ChangeClicksPerSecond(int clkpersec);
	static click_t getClicksForTimestamp(Timestamp timestamp);

	Scheduler(PaletteHost* p);
	~Scheduler();

	void SetRunning(bool b) {
		m_running = b;
	}
	void Stop();
	void AdvanceTime(Timestamp timestamp);
	void RunEveryMillisecondOrSo(Timestamp timestamp);
	std::string DebugString();

	PaletteHost* paletteHost() { return _paletteHost; }

private:
	PaletteHost* _paletteHost;

	int TryLockCallback() {
		return NosuchTryLock(&_callback_mutex,"callback");
	}
	void LockCallback() {
		NosuchLock(&_callback_mutex,"callback");
	}
	void UnlockCallback() {
		NosuchUnlock(&_callback_mutex,"callback");
	}

	bool m_running;

	pthread_mutex_t _callback_mutex;
};

#endif
