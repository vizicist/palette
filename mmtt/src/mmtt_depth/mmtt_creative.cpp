/*
	Space Manifold - a variety of tools for depth cameras and FreeFrame

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

#include "NosuchDebug.h"
#include "mmtt.h"

#ifdef CREATIVE_CAMERA

#include "mmtt_creative.h"

using namespace std;

CreativeDepthCamera::CreativeDepthCamera(MmttServer* s) {
	_server = s;
}

bool CreativeDepthCamera::InitializeCamera()
{
	NosuchDebug("InitializeCamera for Creative");
	pxcStatus sts = PXCSession_Create(&_session);
	if ( sts < PXC_STATUS_NO_ERROR ) {
		NosuchDebug("Failed to create an SDK session for Creative camera");
		return false;
	}

    _capture = new UtilCaptureFile(_session, NULL, false);
    
    PXCCapture::VideoStream::DataDesc request; 
    memset(&request, 0, sizeof(request));
    request.streams[0].format=PXCImage::COLOR_FORMAT_DEPTH;
    // request.streams[1].format=PXCImage::COLOR_FORMAT_DEPTH;
    sts = _capture->LocateStreams (&request); 
    if (sts<PXC_STATUS_NO_ERROR) {
        NosuchDebug("Failed to locate depth stream(s)!!");
        return false;
    }

    for (int idx=0;;idx++) {
		NosuchDebug("InitializeCamera doing queryvideostream for idx=%d",idx);
        PXCCapture::VideoStream *stream_v=_capture->QueryVideoStream(idx);
        if (!stream_v) break;
		NosuchDebug("queryvideostream for idx=%d FOUND ONE!",idx);

        PXCCapture::VideoStream::ProfileInfo pinfo;
        stream_v->QueryProfile(&pinfo);
        WCHAR stream_name[256];
        switch (pinfo.imageInfo.format&PXCImage::IMAGE_TYPE_MASK) {
        case PXCImage::IMAGE_TYPE_COLOR: 
            swprintf_s<256>(stream_name, L"Stream#%d (Color) %dx%d", idx, pinfo.imageInfo.width, pinfo.imageInfo.height);
            break;
        case PXCImage::IMAGE_TYPE_DEPTH: 
            swprintf_s<256>(stream_name, L"Stream#%d (Depth) %dx%d", idx, pinfo.imageInfo.width, pinfo.imageInfo.height);
            break;
        }
        _streams.push_back(stream_v);
    }

    int i;
	_nstreams = (int)_streams.size();
	NosuchDebug("_nstreams = %d",_nstreams);

	_sps = new PXCSmartSPArray(_nstreams);
	_image = new PXCSmartArray<PXCImage>(_nstreams);

    for (i=0;i<_nstreams;i++) { 
        sts=_streams[i]->ReadStreamAsync (&(*_image)[i], &(*_sps)[i]); 
        // if (sts>=PXC_STATUS_NO_ERROR) _nwindows++;
    }

	return true;
}

void CreativeDepthCamera::ProcessDepth() {
	int nframes = 50000;
	pxcStatus sts;

	pxcU32 sidx=0;

#ifdef DO_EXTRA_SYNC
	sts = (*_sps).SynchronizeEx(&sidx,1);  // timeout is 1 millisecond
	if ( sts == PXC_STATUS_EXEC_TIMEOUT ) {
		return;
	}
	if (sts<PXC_STATUS_NO_ERROR ) {
		NosuchDebug("SynchronizeEx(sidx) returned %d",sts);
		return; // break;
	}
#endif

	// loop through all active streams as SynchronizeEx only returns the first active
    for (int j=(int)sidx;j<_nstreams;j++) {
        if (!(*_sps)[j]) {
			continue;
		}
		pxcStatus status = (*_sps)[j]->Synchronize(500);
		if ( status == PXC_STATUS_EXEC_INPROGRESS ) {
			NosuchDebug("synchronize inprogress");
			continue;
		}
		if ( status == PXC_STATUS_EXEC_TIMEOUT ) {
			// NosuchDebug("synchronize timeout");
			continue;
		}
		if ( status != PXC_STATUS_NO_ERROR ) {
			NosuchDebug("synchronize(0) error=%d",status);
			continue;
		}
	    (*_sps).ReleaseRef(j);

		PXCImage* pi = (*_image)[j];
		PXCImage::ImageInfo info;
		sts = pi->QueryInfo(&info);
		if ( sts == PXC_STATUS_NO_ERROR ) {
			// NosuchDebug("info format=%d  w=%d h=%d  _nframes=%d",info.format,info.width,info.height,_nframes);
			PXCImage::ImageData data;
			sts = pi->TryAccess(PXCImage::ACCESS_READ,PXCImage::COLOR_FORMAT_DEPTH,&data);
			if ( sts == PXC_STATUS_NO_ERROR ) {
				pxcBYTE* dplane = data.planes[0];
				pxcBYTE* confidence = data.planes[1];
				processRawDepth((pxcU16*)dplane,(pxcU16*)confidence);
				pi->ReleaseAccess(&data);
			} else {
				NosuchDebug("sts from TryAccess = %d",sts);
			}
		}
		// NosuchDebug("pi=%ld",(long)pi);
        sts=_streams[j]->ReadStreamAsync((*_image).ReleaseRef(j), &(*_sps)[j]);
        if (sts<PXC_STATUS_NO_ERROR) {
			(*_sps)[j]=0;
		}
    }
#ifdef DO_EXTRA_SYNC
    sts = (*_sps).SynchronizeEx();
	if (sts<PXC_STATUS_NO_ERROR) {
		NosuchDebug("SynchronizeEx() returned %d",sts);
		return; // break;
	}
#endif
}

void
CreativeDepthCamera::processRawDepth(pxcU16 *depth,pxcU16* confidence)
{
	uint16_t *depthmm = server()->depthmm_mid;
	uint8_t *depthbytes = server()->depth_mid;
	uint8_t *threshbytes = server()->thresh_mid;

	int i = 0;

	bool filterdepth = ! server()->val_showrawdepth.internal_value;

	// XXX - THIS NEEDS OPTIMIZATION!

	const pxcU16 *pdepth = depth;
	const pxcU16 *pconfidence = confidence;

	int h = height();
	int w = width();
	for (int y=0; y<h; y++) {
	  float backval = (float)(server()->val_backtop.internal_value
		  + (server()->val_backbottom.internal_value - server()->val_backtop.internal_value)
		  * (float(y)/h));

	  pxcU16* lastpixel = depth + (y+1)*w - 1;
	  pxcU16* lastconf = confidence + (y+1)*w - 1;
	  for (int x=0; x<w; x++,i++) {
		// uint16_t mm = *(pdepth++);
		uint16_t mm = *(lastpixel--);
		uint16_t conf = *(lastconf--);
		// conf value is 0-3000?
		// conf = conf / 16;

		depthmm[i] = mm;
#define OUT_OF_BOUNDS 99999
		int deltamm;
		int pval = 0;
		if ( filterdepth ) {
			if ( mm == 0 || mm < server()->val_front.internal_value || mm > backval ) {
				pval = OUT_OF_BOUNDS;
			} else if ( x < server()->val_left.internal_value || x > server()->val_right.internal_value || y < server()->val_top.internal_value || y > server()->val_bottom.internal_value ) {
				pval = OUT_OF_BOUNDS;
			} else if ( conf < server()->val_confidence.internal_value ) {
				pval = OUT_OF_BOUNDS;
			} else {
				// deltamm = (int)val_backtop.internal_value - mm;
				deltamm = (int)backval - mm;
			}
		} else {
			deltamm = (int)backval - mm;
		}
		if ( pval == OUT_OF_BOUNDS || (pval>>8) > 5 ) {
			// black
			*depthbytes++ = 0;
			*depthbytes++ = 0;
			*depthbytes++ = 0;
			threshbytes[i] = 0;
		} else {
			// white
			uint8_t mid = 255 - (deltamm/10);  // a little grey-level based on distance
			*depthbytes++ = mid;
			*depthbytes++ = mid;
			*depthbytes++ = mid;
			threshbytes[i] = 255;
		}
	  }
	}
}

void CreativeDepthCamera::Update() {
	ProcessDepth();
};

void CreativeDepthCamera::Shutdown() {
};

#endif