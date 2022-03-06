#include <iostream>

// include depthai library
// #include <depthai/depthai.hpp>

#include <opencv2/core.hpp>
#include <opencv2/videoio.hpp>
#include <opencv2/highgui.hpp>
#include <opencv2/imgproc.hpp>

extern cv::Mat getCvFrame(std::shared_ptr<dai::ImgFrame> imgframe);

extern "C" {

#include "../include/depthlib.h"

int DepthRunning = 0;
int DepthStopme = 0;

DEPTHLIB_API int DepthIsRunning() {
	return DepthRunning;
}

DEPTHLIB_API void DepthStop()
{
	DepthStopme = 1;
}

DEPTHLIB_API int DepthSet(char *name, char *value)
{
	return 0;
}

DEPTHLIB_API int DepthRun(DepthCallbackFunc callback, int show) {

	DepthRunning = 1;

	// Create pipeline
	dai::Pipeline pipeline;

	// Define sources and outputs
	auto monoLeft = pipeline.create<dai::node::MonoCamera>();
	auto monoRight = pipeline.create<dai::node::MonoCamera>();
	auto depth = pipeline.create<dai::node::StereoDepth>();
	auto xout = pipeline.create<dai::node::XLinkOut>();

	xout->setStreamName("disparity");

	// Properties
	monoLeft->setResolution(dai::MonoCameraProperties::SensorResolution::THE_400_P);
	monoLeft->setBoardSocket(dai::CameraBoardSocket::LEFT);
	monoRight->setResolution(dai::MonoCameraProperties::SensorResolution::THE_400_P);
	monoRight->setBoardSocket(dai::CameraBoardSocket::RIGHT);

	// Create a node that will produce the depth map (using disparity output as it's easier to visualize depth this way)
	depth->initialConfig.setConfidenceThreshold(200); // orig was 245

	// Options: MEDIAN_OFF, KERNEL_3x3, KERNEL_5x5, KERNEL_7x7 (default)
	depth->initialConfig.setMedianFilter(dai::MedianFilter::KERNEL_7x7);

	// Better handling for occlusions:
	depth->setLeftRightCheck(true);

	// Closer-in minimum depth, disparity range is doubled (from 95 to 190):
	// DO NOT CHANGE!
	depth->setExtendedDisparity(false);

	// Better accuracy for longer distance, fractional disparity 32-levels:
	depth->setSubpixel(true);

	// Linking
	monoLeft->out.link(depth->left);
	monoRight->out.link(depth->right);
	depth->disparity.link(xout->input);

	// Connect to device and start pipeline
	dai::Device device(pipeline);

	// Output queue will be used to get the disparity frames from the outputs defined above
	// auto q = device.getOutputQueue("disparity", 4, false);
	auto q = device.getOutputQueue("disparity", 4, false);
	// auto qconfidence = device.getOutputQueue("confidenceMap", 4, false);
	int cnt = 0;
	while (true) {

		try
		{
			auto inDepth = q->get<dai::ImgFrame>();
			auto cvimg = getCvFrame(inDepth);
			// cv::imshow("original", cvimg);

			// auto inConfidence = qconfidence->get<dai::ImgFrame>();
			// auto confimg = getCvFrame(inDepth);
			// cv::imshow("confidence", confimg);

			// Normalization for better visualization
			float ff = 255 / depth->initialConfig.getMaxDisparity();
			cvimg.convertTo(cvimg, CV_8UC1, ff);
			// cv::imshow("normalized", cvimg);

			// cv::blur(cvimg, cvimg, cv::Size(6,6));
			// cv::imshow("blur6", cvimg);

			int x, y;
			for ( x = 0; x< cvimg.cols; x++ ) {
				for (y = 0; y < cvimg.rows; y++) {
					uchar *p = cvimg.data + (y*cvimg.cols + x);
					if (*p > 0 && *p <= 100) {
						// *p = 0;
					}
					else {
						*p = 0;
					}
				}
			}

			// cv::imshow("mycut", cvimg);

			// findblobs
			cv::Mat im_blobs;
			cv::Mat im_keypoints;
			im_blobs = cvimg.clone();

			cv::SimpleBlobDetector::Params params;
			params.filterByArea = true;
			params.minArea = 1200.0;
			// params.maxArea = 1500.0;
			params.filterByCircularity = false;
			params.filterByConvexity = false;
			params.filterByInertia = false;

			params.filterByColor = true;
			params.blobColor = 255;

			params.minDistBetweenBlobs = 50;

			params.minThreshold = 50; //  50;
			params.maxThreshold = 220; //  220;
			params.thresholdStep = 10; // 10;

			cv::Ptr<cv::SimpleBlobDetector> detector = cv::SimpleBlobDetector::create(params);
			std::vector<cv::KeyPoint> keypoints;
			detector->detect(im_blobs, keypoints);
			drawKeypoints(im_blobs, keypoints, im_keypoints, cv::Scalar(0, 0, 255), cv::DrawMatchesFlags::DRAW_RICH_KEYPOINTS);
			for (int ii = 0; ii < keypoints.size(); ii++) {
				int x = int(keypoints[ii].pt.x);
				int y = int(keypoints[ii].pt.y);
				uchar *pp = cvimg.data + (y*cvimg.cols + x);
				if ( *pp != 0 ) {
					char msg[256];
					sprintf_s(msg, sizeof(msg), "\"x\":%f,\"y\":%f,\"z\":%d,\"size\":\"%f\"",
						keypoints[ii].pt.x, keypoints[ii].pt.y, *pp, keypoints[ii].size);
					callback((char*)"depth", msg);
				}
			}
			if (show) {
				cv::imshow("depth keypoints", im_keypoints);
			}

			// cv::Mat im_thresh;
			// cv::threshold(cvimg, im_thresh, 100, 110, cv::THRESH_BINARY);
			// cv::imshow("threshold", im_thresh);
		}
		catch (...)
		{
			printf("Hey! Exception caught in depthRun()?\n");
		}

		int key = cv::waitKey(1);
		if (key == 'q' || key == 'Q') {
			break;
		}
	}
	DepthRunning = 0;
	printf("depthRun returning\n");
	return 0;
}

}
