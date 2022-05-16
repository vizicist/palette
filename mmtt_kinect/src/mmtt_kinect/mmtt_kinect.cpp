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

#include "stdafx.h"
#include "NosuchDebug.h"
#include "mmtt.h"
#include "mmtt_kinect.h"

using namespace std;

KinectDepthCamera::KinectDepthCamera(MmttServer* s) {
	_server = s;
}

bool KinectDepthCamera::InitializeCamera() {

    INuiSensor * sensor;
    HRESULT hr;

	for ( int i=0; i<MAX_DEPTH_CAMERAS; i++ ) {
	    m_hNextDepthFrameEvent[i] = INVALID_HANDLE_VALUE;
	    m_pDepthStreamHandle[i] = INVALID_HANDLE_VALUE;
	    m_pNuiSensor[i] = NULL;
	}

    int iSensorCount = 0;
    hr = NuiGetSensorCount(&iSensorCount);
    if (FAILED(hr)) {
        return false;
    }

    // Look at each Kinect sensor
	m_nsensors = 0;
	m_currentSensor = -1;
    for (int i = 0; i < iSensorCount; ++i) {
        // Create the sensor so we can check status, if we can't create it, move on to the next
        hr = NuiCreateSensorByIndex(i, &sensor);
        if (FAILED(hr)) {
            continue;
        }

        // Get the status of the sensor, and if connected, then we can initialize it
        hr = sensor->NuiStatus();
        if (S_OK == hr) {
	        hr = sensor->NuiInitialize(NUI_INITIALIZE_FLAG_USES_DEPTH); 
	        if (SUCCEEDED(hr)) {
	            // Create an event that will be signaled when depth data is available
	            m_hNextDepthFrameEvent[m_nsensors] = CreateEvent(NULL, TRUE, FALSE, NULL);

	            // Open a depth image stream to receive depth frames
	            hr = sensor->NuiImageStreamOpen(
	                NUI_IMAGE_TYPE_DEPTH,
	                NUI_IMAGE_RESOLUTION_640x480,
	                0,
	                2,
	                m_hNextDepthFrameEvent[m_nsensors],
	                &(m_pDepthStreamHandle[m_nsensors]));
				if ( SUCCEEDED(hr) ) {
		            m_pNuiSensor[m_nsensors] = sensor;
					m_nsensors++;
				}
	        }
        } else {
	        // This sensor wasn't OK, so release it since we're not using it
	        sensor->Release();
		}
    }

    if ( m_nsensors == 0 ) {
        NosuchDebug("No ready Kinect found!");
        return false;
    }
	m_currentSensor = 0;
	return true;
}

void KinectDepthCamera::ProcessDepth()
{
    HRESULT hr;
    NUI_IMAGE_FRAME imageFrame;

    // Attempt to get the depth frame
    hr = m_pNuiSensor[m_currentSensor]->NuiImageStreamGetNextFrame(m_pDepthStreamHandle[m_currentSensor], 0, &imageFrame);
    if (FAILED(hr))
    {
        return;
    }

    BOOL nearMode;
    INuiFrameTexture* pTexture;

    // Get the depth image pixel texture
    hr = m_pNuiSensor[m_currentSensor]->NuiImageFrameGetDepthImagePixelFrameTexture(
        m_pDepthStreamHandle[m_currentSensor], &imageFrame, &nearMode, &pTexture);
    if (FAILED(hr))
    {
        goto ReleaseFrame;
    }

    NUI_LOCKED_RECT LockedRect;

    // Lock the frame data so the Kinect knows not to modify it while we're reading it
    pTexture->LockRect(0, &LockedRect, NULL, 0);

    // Make sure we've received valid data
    if (LockedRect.Pitch != 0)
    {
        // Get the min and max reliable depth for the current frame
        int minDepth = (nearMode ? NUI_IMAGE_DEPTH_MINIMUM_NEAR_MODE : NUI_IMAGE_DEPTH_MINIMUM) >> NUI_IMAGE_PLAYER_INDEX_SHIFT;
        int maxDepth = (nearMode ? NUI_IMAGE_DEPTH_MAXIMUM_NEAR_MODE : NUI_IMAGE_DEPTH_MAXIMUM) >> NUI_IMAGE_PLAYER_INDEX_SHIFT;

        const NUI_DEPTH_IMAGE_PIXEL * pBufferRun = reinterpret_cast<const NUI_DEPTH_IMAGE_PIXEL *>(LockedRect.pBits);

        // end pixel is start + width*height - 1
        const NUI_DEPTH_IMAGE_PIXEL * pBufferEnd = pBufferRun + (width() * height());

		// mmtt_server_process_rawdepth(pBufferRun);
		processRawDepth(pBufferRun);
    }

    // We're done with the texture so unlock it
    pTexture->UnlockRect(0);

    pTexture->Release();

ReleaseFrame:
    // Release the frame
    m_pNuiSensor[m_currentSensor]->NuiImageStreamReleaseFrame(m_pDepthStreamHandle[m_currentSensor], &imageFrame);
}

void
KinectDepthCamera::processRawDepth(const NUI_DEPTH_IMAGE_PIXEL *depth)
{
	uint16_t *depthmm = server()->depthmm_mid;
	uint8_t *depthbytes = server()->depth_mid;
	uint8_t *threshbytes = server()->thresh_mid;

	int i = 0;

	bool filterdepth = ! server()->val_showrawdepth.internal_value;

	// XXX - THIS NEEDS OPTIMIZATION!

	const NUI_DEPTH_IMAGE_PIXEL *pdepth = depth;

	int h = height();
	int w = width();
	for (int y=0; y<h; y++) {
	  float backval = (float)(server()->val_backtop.internal_value
		  + (server()->val_backbottom.internal_value - server()->val_backtop.internal_value)
		  * (float(y)/h));

	  for (int x=0; x<w; x++,i++) {
		// int d = depth[i].depth;
		USHORT d = (pdepth++)->depth;

		// This is a bottleneck
#ifdef USE_NUIDEPTHPIXELTODEPTH
		uint16_t mm = NuiDepthPixelToDepth(d);
		// Not totally sure why, but I think this is needed due to the 3-bit shift for the player index?
		// Otherwise, the millimeter values don't seem to make sense.
		mm *= 8;
#else
		uint16_t mm = d & (~07);	// just zero out the first 3 bits, much more efficient
#endif

		depthmm[i] = mm;
#define OUT_OF_BOUNDS 99999
		int deltamm;
		int pval = 0;
		if ( filterdepth ) {
			if ( mm == 0 || mm < server()->val_front.internal_value || mm > backval ) {
				pval = OUT_OF_BOUNDS;
			} else if ( x < server()->val_left.internal_value || x > server()->val_right.internal_value || y < server()->val_top.internal_value || y > server()->val_bottom.internal_value ) {
				pval = OUT_OF_BOUNDS;
			} else {
				// deltamm = (int)server()->val_backtop.internal_value - mm;
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

void KinectDepthCamera::Update() {

    hEvents[0] = m_hNextDepthFrameEvent[m_currentSensor];

    // Check to see if we have either a message (by passing in QS_ALLINPUT)
    // Or a depth event (hEvents)
    // Update() will check for depth events individually, in case more than one are signalled
    DWORD dwEvent = MsgWaitForMultipleObjects(eventCount, hEvents, FALSE, INFINITE, QS_ALLINPUT);

    // Check if this is an event we're waiting on and not a timeout or message
    if (WAIT_OBJECT_0 == dwEvent) {
		if (NULL == m_pNuiSensor[m_currentSensor]) {
		        return;
	    }
		if ( WAIT_OBJECT_0 == WaitForSingleObject(m_hNextDepthFrameEvent[m_currentSensor], 0) ) {
		        ProcessDepth();
		}
    }
}

void KinectDepthCamera::Shutdown() {
    if (m_pNuiSensor[m_currentSensor])
    {
        m_pNuiSensor[m_currentSensor]->NuiShutdown();
    }

    if (m_hNextDepthFrameEvent[m_currentSensor] != INVALID_HANDLE_VALUE)
    {
        CloseHandle(m_hNextDepthFrameEvent[m_currentSensor]);
    }

    SafeRelease(m_pNuiSensor[m_currentSensor]);
}