#include <iostream>

extern "C" {
#include "depthlib.h"

void DepthCallback(char *subj, char *msg) {
	printf("depthRun callback subj=%s msg=%s\n", subj, msg);
}

int main() {
	int r = DepthRun(DepthCallback,1);
	printf("DepthRun returned?  r=%d\n", r);
	return 0;
}

}
