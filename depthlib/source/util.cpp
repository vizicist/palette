// include depthai library
#include <depthai/depthai.hpp>

#include <opencv2/core.hpp>
#include <opencv2/videoio.hpp>
#include <opencv2/highgui.hpp>
#include <opencv2/imgproc.hpp>

void oldfindblobs(cv::Mat im) {
#define NEWBLOB
#ifdef NEWBLOB
	cv::SimpleBlobDetector::Params params;
	cv::Ptr<cv::SimpleBlobDetector> detector = cv::SimpleBlobDetector::create(params);
	std::vector<cv::KeyPoint> keypoints;
	detector->detect(im, keypoints);
	drawKeypoints(im, keypoints, im, cv::Scalar(0, 0, 255), cv::DrawMatchesFlags::DRAW_RICH_KEYPOINTS);
	// cv::imshow("crop", im);
#else
	// Detect blobs.
	std::vector<cv::KeyPoint> keypoints;
	imshow("findblobs2", im);
	detector->detect(im, keypoints);
	// Draw detected blobs as red circles.
	// DrawMatchesFlags::DRAW_RICH_KEYPOINTS flag ensures the size of the circle corresponds to the size of blob
	cv::Mat im_with_keypoints;
	drawKeypoints(im, keypoints, im_with_keypoints, cv::Scalar(0, 0, 255), cv::DrawMatchesFlags::DRAW_RICH_KEYPOINTS);
#endif
}

cv::Mat getCvMat(std::shared_ptr<dai::ImgFrame> imgframe, bool deepCopy) {

	// Convert to cv::Mat. If deepCopy enabled, then copy pixel data, otherwise reference only

	cv::Mat mat;
	cv::Size size = { 0, 0 };
	int type = 0;

	switch (imgframe->getType()) {
	case dai::RawImgFrame::Type::RGB888i:
	case dai::RawImgFrame::Type::BGR888i:
	case dai::RawImgFrame::Type::BGR888p:
	case dai::RawImgFrame::Type::RGB888p:
		size = cv::Size(imgframe->getWidth(), imgframe->getHeight());
		type = CV_8UC3;
		break;

	case dai::RawImgFrame::Type::YUV420p:
	case dai::RawImgFrame::Type::NV12:
	case dai::RawImgFrame::Type::NV21:
		size = cv::Size(imgframe->getWidth(), imgframe->getHeight() * 3 / 2);
		type = CV_8UC1;
		break;

	case dai::RawImgFrame::Type::RAW8:
	case dai::RawImgFrame::Type::GRAY8:
		size = cv::Size(imgframe->getWidth(), imgframe->getHeight());
		type = CV_8UC1;
		break;

	case dai::RawImgFrame::Type::GRAYF16:
		size = cv::Size(imgframe->getWidth(), imgframe->getHeight());
		type = CV_16FC1;
		break;

	case dai::RawImgFrame::Type::RAW16:
		size = cv::Size(imgframe->getWidth(), imgframe->getHeight());
		type = CV_16UC1;
		break;

	case dai::RawImgFrame::Type::RGBF16F16F16i:
	case dai::RawImgFrame::Type::BGRF16F16F16i:
	case dai::RawImgFrame::Type::RGBF16F16F16p:
	case dai::RawImgFrame::Type::BGRF16F16F16p:
		size = cv::Size(imgframe->getWidth(), imgframe->getHeight());
		type = CV_16FC3;
		break;

	case dai::RawImgFrame::Type::BITSTREAM:
	default:
		size = cv::Size(static_cast<int>(imgframe->getData().size()), 1);
		type = CV_8UC1;
		break;
	}

	// Check if enough data
	long requiredSize = CV_ELEM_SIZE(type) * size.area();
	long actualSize = static_cast<long>(imgframe->getData().size());
	if (actualSize < requiredSize) {
		throw std::runtime_error("ImgFrame doesn't have enough data to encode specified frame, required " + std::to_string(requiredSize) + ", actual "
			+ std::to_string(actualSize) + ". Maybe metadataOnly transfer was made?");
	}
	else if (actualSize > requiredSize) {
		// FIXME doesn't build on Windows (multiple definitions during link)
		// spdlog::warn("ImgFrame has excess data: actual {}, expected {}", actualSize, requiredSize);
	}
	if (imgframe->getWidth() <= 0 || imgframe->getHeight() <= 0) {
		throw std::runtime_error("ImgFrame metadata not valid (width or height = 0)");
	}

	// Copy or reference to existing data
	if (deepCopy) {
		// Create new image data
		mat.create(size, type);
		// Copy number of bytes that are available by Mat space or by img data size
		std::memcpy(mat.data, imgframe->getData().data(), std::min((long)(imgframe->getData().size()), (long)(mat.dataend - mat.datastart)));
	}
	else {
		mat = cv::Mat(size, type, imgframe->getData().data());
	}

	return mat;
}

cv::Mat getCvFrame(std::shared_ptr<dai::ImgFrame> imgframe) {
	// cv::Mat getFrame(std::shared_ptr<dai::ImgFrame> imgframe, bool deepCopy) {
	cv::Mat frame = getCvMat(imgframe, true);
	cv::Mat output;

	switch (imgframe->getType()) {
	case dai::RawImgFrame::Type::RGB888i:
		cv::cvtColor(frame, output, cv::ColorConversionCodes::COLOR_RGB2BGR);
		break;

	case dai::RawImgFrame::Type::BGR888i:
		output = frame.clone();
		break;

	case dai::RawImgFrame::Type::RGB888p: {
		cv::Size s(imgframe->getWidth(), imgframe->getHeight());
		std::vector<cv::Mat> channels;
		// RGB
		channels.push_back(cv::Mat(s, CV_8UC1, imgframe->getData().data() + s.area() * 2));
		channels.push_back(cv::Mat(s, CV_8UC1, imgframe->getData().data() + s.area() * 1));
		channels.push_back(cv::Mat(s, CV_8UC1, imgframe->getData().data() + s.area() * 0));
		cv::merge(channels, output);
	} break;

	case dai::RawImgFrame::Type::BGR888p: {
		cv::Size s(imgframe->getWidth(), imgframe->getHeight());
		std::vector<cv::Mat> channels;
		// BGR
		channels.push_back(cv::Mat(s, CV_8UC1, imgframe->getData().data() + s.area() * 0));
		channels.push_back(cv::Mat(s, CV_8UC1, imgframe->getData().data() + s.area() * 1));
		channels.push_back(cv::Mat(s, CV_8UC1, imgframe->getData().data() + s.area() * 2));
		cv::merge(channels, output);
	} break;

	case dai::RawImgFrame::Type::YUV420p:
		cv::cvtColor(frame, output, cv::ColorConversionCodes::COLOR_YUV2BGR_IYUV);
		break;

	case dai::RawImgFrame::Type::NV12:
		cv::cvtColor(frame, output, cv::ColorConversionCodes::COLOR_YUV2BGR_NV12);
		break;

	case dai::RawImgFrame::Type::NV21:
		cv::cvtColor(frame, output, cv::ColorConversionCodes::COLOR_YUV2BGR_NV21);
		break;

	case dai::RawImgFrame::Type::RAW8:
	case dai::RawImgFrame::Type::RAW16:
	case dai::RawImgFrame::Type::GRAY8:
	case dai::RawImgFrame::Type::GRAYF16:
		output = frame.clone();
		break;

	default:
		output = frame.clone();
		break;
	}

	return output;
}