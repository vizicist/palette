/*
******************************************************************************

tstring versions 1.0, Octover 25, 2006 is Copyright (c) 2004 Emmanuel 
Deloget and is distributed according to the same disclaimer and license 
as noted in the tstring.h file

with the following additions to the disclaimer:

   There is no warranty against interference with your enjoyment of the
   library or against infringement.  There is no warranty that our
   efforts or the library will fulfill any of your particular purposes
   or needs.  This library is provided with all faults, and the entire
   risk of satisfactory quality, performance, accuracy, and effort is with
   the user.

The tstring library is supplied "AS IS".  The Contributing Authors 
disclaim all warranties, expressed or implied, including, without 
limitation, the warranties of merchantability and of fitness 
for any purpose.  The Contributing Authors assume no liability for 
direct, indirect, incidental, special, exemplary, or consequential 
damages, which may result from the use of the tstring library, 
even if advised of the possibility of such damage.

Permission is hereby granted to use, copy, modify, and distribute this
source code, or portions hereof, for any purpose, without fee, subject
to the following restrictions:

1. The origin of this source code must not be misrepresented.

2. Altered versions must be plainly marked as such and must not
   be misrepresented as being the original source.

3. This Copyright notice may not be removed or altered from any
   source or altered source distribution.

******************************************************************************
*/
#ifndef TSTD_TSTRING_H
#define TSTD_TSTRING_H

#include <string>
#include <iostream>
#include <fstream>

namespace tstd
{

#ifdef UNICODE

	// ------------------------------
	// ------------------------------ UNICODE SUPPORT
	// ------------------------------

#ifndef TEXT
#	define TEXT(s)		L##s
#endif
#ifndef _T
#	define _T(s)		L##s
#endif

	typedef std::wstring			tstring;
	typedef std::wostream			tostream;
	typedef std::wistream			tistream;
	typedef std::wiostream			tiostream;
	typedef std::wistringstream		tistringstream;
	typedef std::wostringstream		tostringstream;
	typedef std::wstringstream		tstringstream;
	typedef std::wifstream			tifstream;
	typedef std::wofstream			tofstream;
	typedef std::wfstream			tfstream;
	typedef std::wfilebuf			tfilebuf;
	typedef std::wios				tios;
	typedef std::wstreambuf			tstreambuf;
	typedef std::wstreampos			tstreampos;
	typedef std::wstringbuf			tstringbuf;

	namespace 
	{
		tostream& tcout = std::wcout;
		tostream& tcerr = std::wcerr;
		tostream& tclog = std::wclog;
		tistream& tcin	= std::wcin;

		tstring wstr_to_tstr(const std::wstring& arg)
		{
			return arg;
		}
		tstring str_to_tstr(const std::string& arg)
		{
			tstring res(arg.length(), L'\0');
			mbstowcs(const_cast<wchar_t*>(res.data()), arg.c_str(), arg.length());
			return res;
		}
		std::wstring tstr_to_wstr(const tstring& arg)
		{
			return arg;
		}
		std::string tstr_to_str(const tstring& arg)
		{
			std::string res(arg.length(), '\0');
			wcstombs(const_cast<char*>(res.data()), arg.c_str(), arg.length());
			return res;
		}
	};

#else

	// ------------------------------
	// ------------------------------ MBCS/SBCS SUPPORT
	// ------------------------------

#ifndef TEXT
#	define TEXT(s)		s
#endif
#ifndef _T
#	define _T(s)		s
#endif


	typedef std::string				tstring;
	typedef std::ostream			tostream;
	typedef std::istream			tistream;
	typedef std::iostream			tiostream;
	typedef std::istringstream		tistringstream;
	typedef std::ostringstream		tostringstream;
	typedef std::stringstream		tstringstream;
	typedef std::ifstream			tifstream;
	typedef std::ofstream			tofstream;
	typedef std::fstream			tfstream;
	typedef std::filebuf			tfilebuf;
	typedef std::ios				tios;
	typedef std::streambuf			tstreambuf;
	typedef std::streampos			tstreampos;
	typedef std::stringbuf			tstringbuf;

	namespace 
	{
		tostream& tcout = std::cout;
		tostream& tcerr = std::cerr;
		tostream& tclog = std::clog;
		tistream& tcin	= std::cin;

		tstring wstr_to_tstr(const std::wstring& arg)
		{
			tstring res(arg.length(), '\0');
			wcstombs(const_cast<char*>(res.data()), arg.c_str(), arg.length());
			return res;
		}
		tstring str_to_tstr(const std::string& arg)
		{
			return arg;
		}
		std::wstring tstr_to_wstr(const tstring& arg)
		{
			std::wstring res(arg.length(), L'\0');
			mbstowcs(const_cast<wchar_t*>(res.data()), arg.c_str(), arg.length());
			return res;
		}
		std::string tstr_to_str(const tstring& arg)
		{
			return arg;
		}
	};


#endif

}

#endif // TSTD_TSTRING_H
