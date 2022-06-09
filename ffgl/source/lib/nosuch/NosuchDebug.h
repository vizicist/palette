#pragma once

#include <string>

extern int NosuchDebugLevel;
extern bool NosuchDebugCursor;
extern bool NosuchDebugAPI;
extern bool NosuchDebugSprite;
extern bool NosuchDebugToLog;
extern bool NosuchDebugTimeTag;
extern bool NosuchDebugAutoFlush;
// extern std::string NosuchDebugLogPath;
extern std::string NosuchDebugPrefix;
extern std::string NosuchAppName;
extern int NosuchDebugTag;

std::string NosuchSnprintf(const char *fmt, ...);

void NosuchDebugSetThreadName(void* p, std::string name);
void NosuchDebugDumpLog();
void NosuchDebug(const char *fmt, ... );
void NosuchDebug(int level, char const *fmt, ... );
void NosuchErrorOutput(const char *fmt, ...);
std::string NosuchForwardSlash(std::string filepath);

#define NosuchAssert(expr) if(!(expr)){ throw std::runtime_error("NosuchAssert exception!\n");}
