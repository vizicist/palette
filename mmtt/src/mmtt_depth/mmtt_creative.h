/*
	Space Manifold - a variety of tools for Kinect and FreeFrame

	Copyright (c) 2011-2012 Tim Thompson <me@timthompson.com>

	Permission is hereby granted, free of charge, to any person obtaining
	a copy of this software and associated documentation files
	(the "Software"), to deal in the Software without restriction,
	including without limitation the rights to use, copy, modify, merge,
	publish, distribute, sublicense, and/or sell copies of the Software,
	and to permit persons to whom the Software is furnished to do so,
	subject to the following conditions:

	The above copyright notice and this permission notice shall be
	included in all copies or substantial portions of the Software.

	Any person wishing to distribute modifications to the Software is
	requested to send the modifications to the original developer so that
	they can be incorporated into the canonical version.

	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
	EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
	MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
	IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR
	ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF
	CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
	WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

#ifndef MMTT_CREATIVE_H
#define MMTT_CREATIVE_H

#include "mmtt.h"

#ifdef CREATIVE_CAMERA

#include <vector>
#include <map>
#include <iostream>

#include "ip/NetworkingUtils.h"

#include "OscSender.h"
#include "OscMessage.h"
#include "BlobResult.h"
#include "blob.h"

#include "NosuchHttp.h"
#include "NosuchException.h"

class UT_SharedMem;
class MMTT_SharedMemHeader;
class TOP_SharedMemHeader;

#include "pxcsession.h"
#include "pxcsmartptr.h"
#include "pxccapture.h"
#include "util_render.h"
#include "util_capture_file.h"
#include "util_cmdline.h"

#include "mmtt_depth.h"

#define _USE_MATH_DEFINES // To get definition of M_PI
#include <math.h>

class CreativeDepthCamera : public DepthCamera {
public:
	CreativeDepthCamera(MmttServer* s);
	~CreativeDepthCamera();
	const int width() { return 320; }
	const int height() { return 240; }
	const int default_backtop() { return 700; };
	const int default_backbottom() { return 700; };
	bool InitializeCamera();
	void Update();
	void Shutdown();

private:

	PXCSmartPtr<PXCSession> _session;
	std::vector<PXCCapture::VideoStream*> _streams;
    PXCSmartSPArray* _sps;
    PXCSmartArray<PXCImage>* _image;
	int _nactive;
	int _nstreams;
	UtilCaptureFile* _capture;

	void ProcessDepth();
	void processRawDepth(pxcU16* depth, pxcU16* confidence);
};
#endif

#endif
