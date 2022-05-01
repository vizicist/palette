package kit

import (
)

/*
#include <windows.h>
#include <malloc.h>
#include <dos.h>
#include <time.h>
#include <io.h>
#include <direct.h>

#include "mdep.h"

void mdep_destroywindow(void);
*/

const SEPARATOR = "\\"

func mdep_hello(argc int, argv []string) {
	// _set_fmode(O_BINARY);
}

func mdep_bye() {
	mdep_destroywindow()
}

func mdep_changedir(d string) {
	return os.Chdir(d)
}

func mdep_currentdir() string {
	return os.Getwd()
}

// #define fileisdir(fd) (((fd).dwFileAttributes & FILE_ATTRIBUTE_DIRECTORY)!=0)

type LsdirCallback func(string,int)
func mdep_lsdir(dir string, exp string, callback LsdirCallback) int {
	/*
	WIN32_FIND_DATA fd;
	HANDLE h;
	char buff[255];	// should be MAX_PATH

	strcpy(buff,dir);
	strcat(buff,SEPARATOR);
	strcat(buff,exp);
	h = FindFirstFile(buff,&fd);
	if ( h == INVALID_HANDLE_VALUE )
		return(0);	// okay, there's just nothing that matches
	callback(fd.cFileName,fileisdir(fd)?1:0);
	while ( FindNextFile(h,&fd) == TRUE )
		callback(fd.cFileName,fileisdir(fd)?1:0);
	return(0);
	*/
	return 0
}

func mdep_filetime(fn string) long {
	/*
	struct _stat s;

	if ( _stat(fn,&s) == -1 )
		return(-1);

	//
	// Win98 (and 95?, probably) returns a -1 in the st_mtime
	// field when then modification time of the file is
	// either way in the past or future (I had a file for
	// which "DIR" said it was 8/7/72 and mks's "ls -l"
	// said it was modified in 2115, so I'm not sure which it
	// was, but it was returning -1 in the st_mtime field.
	//
	if ( s.st_mtime < 0 )
		s.st_mtime = 0;

	return((long)(s.st_mtime));
	*/
	return 0
}

int
mdep_fisatty(FILE *f)
{
	if ( f == stdin )
		return 1;
	else
		return _isatty(_fileno(f));
}

long
mdep_currtime(void)
{
	SYSTEMTIME st;
	GetSystemTime(&st);
	return (
		(3600*24*30)*st.wMonth
		+ (3600*24)*st.wDay
		+ 3600*st.wHour
		+ 60*st.wMinute
		+ st.wSecond);
}

long
mdep_coreleft(void)
{
	MEMORYSTATUS m;
	m.dwLength = sizeof(MEMORYSTATUS);
	GlobalMemoryStatus(&m);
	return (long) m.dwAvailPageFile;
}

int
mdep_full_or_relative_path(char *path)
{
	if ( *path == '/'
		|| *path=='\\'
		|| *(path+1)==':'
		|| *path == '.' )
		return 1;
	else
		return 0;
}

int
mdep_makepath(char *dirname, char *filename, char *result, int resultsize)
{
	char *p;

	if ( resultsize < (int)(strlen(dirname)+strlen(filename)+5) )
		return 1;

	/* special case for current directory, */
	/* since ./file doesn't always work? */
	if ( strcmp(dirname,".") == 0 ) {
		strcpy(result,filename);
		return 0;
	}

	strcpy(result,dirname);
	if ( *dirname != '\0' )
		strcat(result,SEPARATOR);
	p = strchr(result,'\0');
	strcat(result,filename);

#ifdef OLDSTUFF
	/* If filename is of form *.*, enforce 8.3 character limits */
	if ( (q=strchr(p,'.')) != NULL ) {
		if ( strlen(q+1) > 3 )
			*(q+4) = '\0';
		if ( (q-p) > 8 )
			strcpy(p+8,q);
	}
#endif
	return 0;
}

#ifdef LOCALUNLINK
int
unlink(const char *path)
{
	return remove(path);
}
#endif
