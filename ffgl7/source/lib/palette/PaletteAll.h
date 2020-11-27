
// typedef long long int64_t;
// typedef unsigned long long uint64_t;
#define PALETTE_HACK

#include "winsock.h"
#include "FFGLPluginSDK.h"
#include "FFGL.h"
#include "FFGLSDK.h"
#include "FFGLLog.h"
#include "glm/glm.hpp"

#include "mmsystem.h"
#include "cJSON.h"
#include "NosuchUtil.h"
#include "NosuchOscInput.h"
#include "NosuchOscUdpInput.h"
#include "osc/OscOutboundPacketStream.h"
#include "PaletteOscInput.h"
#include "NosuchGraphics.h"
#include "NosuchException.h"
#include "Params.h"
#include "PaletteUtil.h"
#include "Region.h"
#include "NosuchColor.h"
#include "Scheduler.h"
#include "TrackedCursor.h"
#include "DrawQuad.h"
#include "DrawTriangle.h"
#include "PaletteDrawer.h"
#include "Palette.h"
#include "Sprite.h"
#include "PaletteHost.h"
