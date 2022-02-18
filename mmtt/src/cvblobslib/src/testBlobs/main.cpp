// main.cpp : Defines the entry point for the console application.
//

// #include "stdafx.h"
#include "cv.h"
#include "highgui.h"
#include <stdio.h>
#include <conio.h>

// Main blob library include
#include "BlobResult.h"

char wndname[] = "Blob Extraction";
char tbarname1[] = "Threshold";
char tbarname2[] = "Blob Size";

// The output and temporary images
IplImage* originalThr = 0;
IplImage* original = 0;
IplImage* displayedImage = 0;

int param1,param2;



// threshold trackbar callback
void on_trackbar( int dummy )
{
	if(!originalThr)
	{
		originalThr = cvCreateImage(cvGetSize(original), IPL_DEPTH_8U,1);
	}

	if(!displayedImage)
	{
		displayedImage = cvCreateImage(cvGetSize(original), IPL_DEPTH_8U,3);
	}
	
	// threshold input image
	cvThreshold( original, originalThr, param1, 255, CV_THRESH_BINARY );

	// get blobs and filter them using its area
	CBlobResult blobs;
	int i;
	CBlob *currentBlob;

	// find blobs in image
	blobs = CBlobResult( originalThr, NULL, 255 );
	blobs.Filter( blobs, B_EXCLUDE, CBlobGetArea(), B_LESS, param2 );

	// display filtered blobs
	cvMerge( originalThr, originalThr, originalThr, NULL, displayedImage );

	for (i = 0; i < blobs.GetNumBlobs(); i++ )
	{
		currentBlob = blobs.GetBlob(i);
		currentBlob->FillBlob( displayedImage, CV_RGB(255,0,0));
	}
	 
    cvShowImage( wndname, displayedImage );
	
}



int main( int argc, char** argv )
{

	param1 = 100;
	param2 = 2000;
	
	// open input image
	original = cvLoadImage("pic6.png",0);

	cvNamedWindow("input");
	cvShowImage("input", original );
	
	cvNamedWindow(wndname, 0);
    cvCreateTrackbar( tbarname1, wndname, &param1, 255, on_trackbar );
	cvCreateTrackbar( tbarname2, wndname, &param2, 30000, on_trackbar );
	
	// Call to update the view
	for(;;)
    {
        int c;
        
        // Call to update the view
        on_trackbar(0);

        c = cvWaitKey(0);

	   if( c == 27 )
            break;
	}
    
    cvReleaseImage( &original );
	cvReleaseImage( &originalThr );
	cvReleaseImage( &displayedImage );
    
    cvDestroyWindow( wndname );
    
    return 0;
}
