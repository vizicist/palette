#include "SpoutSender.h"
#include "SpoutReceiver.h"
#include <stdlib.h>

extern "C" {

typedef void* GoSpoutSender;
SpoutReceiver r;

GoSpoutSender GoCreateSender(const char* sendername, int width, int height) {
	SpoutSender* s = new SpoutSender;
	s->CreateSender(sendername,width,height);
	GoSpoutSender gss = (GoSpoutSender)s;
	return gss;
}

bool GoSendTexture(GoSpoutSender gss, unsigned int texture, int width, int height) {
	SpoutSender* s = (SpoutSender*)gss;
	return s->SendTexture(texture,GL_TEXTURE_2D,width,height);
}

bool GoCreateReceiver(char* sendername, unsigned int* width, unsigned int *height, bool bUseActive) {
	bool b;
	unsigned int w = *width;
	unsigned int h = *height;
	b = r.CreateReceiver(sendername,w,h,bUseActive);
	if ( b ) {
		*width = w;
		*height = h;
	}
	return b;
}

void GoReleaseReceiver() {
	r.ReleaseReceiver();
}

bool GoReceiveTexture(char* sendername, unsigned int* width, unsigned int *height, int textureID, int textureTarget, bool bInvert, int hostFBO) {
	bool b;
	unsigned int w = *width;
	unsigned int h = *height;
	b = r.ReceiveTexture(sendername,w,h,textureID,textureTarget,bInvert,hostFBO);
	if ( b ) {
		*width = w;
		*height = h;
	}
	return b;
}

}
