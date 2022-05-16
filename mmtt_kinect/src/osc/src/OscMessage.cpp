/*
 Copyright (c) 2010, Hector Sanchez-Pajares
 Aer Studio http://www.aerstudio.com
 All rights reserved.
 
 
 This is a block for OSC Integration for the Cinder framework (http://libcinder.org)
 
 Redistribution and use in source and binary forms, with or without modification, are permitted provided that
 the following conditions are met:
 
 * Redistributions of source code must retain the above copyright notice, this list of conditions and
 the following disclaimer.
 * Redistributions in binary form must reproduce the above copyright notice, this list of conditions and
 the following disclaimer in the documentation and/or other materials provided with the distribution.
 
 THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY EXPRESS OR IMPLIED
 WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A
 PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR
 ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED
 TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION)
 HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING
 NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
 POSSIBILITY OF SUCH DAMAGE.
 */

#include "OscMessage.h"

OscMessage::~OscMessage(){
	clear();
}

void OscMessage::clear(){
	for (unsigned int i=0; i < args.size(); ++i){
		delete args[i];
	}
	args.clear();
	address = "";
}

int OscMessage::getNumArgs() const{
	return (int)args.size();
}

ArgType OscMessage::getArgType(int index) const{
	if (index >= (int)args.size()){
		throw OscExcOutOfBounds();
	}else {
		return args[index]->getType();
	}
}

std::string OscMessage::getArgTypeName(int index) const{
	if (index >= (int)args.size()){
		throw OscExcOutOfBounds();
	}else {
		return args[index]->getTypeName();
	}
}

int32_t OscMessage::getArgAsInt32(int index, bool typeConvert) const{
	if (getArgType(index) != TYPE_INT32){
		if( typeConvert && (getArgType(index) == TYPE_FLOAT) )
			return (int32_t)((ArgFloat*)args[index])->get();
		else
			throw OscExcInvalidArgumentType();
	}else 
		return ((ArgInt32*)args[index])->get();
}

float OscMessage::getArgAsFloat(int index, bool typeConvert) const{
	if (getArgType(index) != TYPE_FLOAT){
		if( typeConvert && (getArgType(index) == TYPE_INT32) )
			return (float)((ArgInt32*)args[index])->get();
		else
			throw OscExcInvalidArgumentType();
	}else
        return ((ArgFloat*)args[index])->get();
}

std::string OscMessage::getArgAsString( int index, bool typeConvert ) const{
    if (getArgType(index) != TYPE_STRING ){
	    if (typeConvert && (getArgType(index) == TYPE_FLOAT) ){
            char buf[1024];
            sprintf(buf,"%f",((ArgFloat*)args[index])->get() );
            return std::string( buf );
        }
	    else if (typeConvert && (getArgType(index) == TYPE_INT32)){
            char buf[1024];
            sprintf(buf,"%i",((ArgInt32*)args[index])->get() );
            return std::string( buf );
        }
        else
            throw OscExcInvalidArgumentType();
	}
	else
        return ((ArgString*)args[index])->get();
}

void OscMessage::addIntArg( int32_t argument ){
	args.push_back( new ArgInt32( argument ) );
}

void OscMessage::addFloatArg( float argument ){
	args.push_back( new ArgFloat( argument ) );
}

void OscMessage::addStringArg( std::string argument ){
	args.push_back( new ArgString( argument ) );
}
	
OscMessage& OscMessage::copy( const OscMessage& other ){

	address = other.address;
	
	remote_host = other.remote_host;
	remote_port = other.remote_port;
	
	for ( int i=0; i<(int)other.args.size(); ++i ){
		ArgType argType = other.getArgType( i );
		if ( argType == TYPE_INT32 )
			args.push_back( new ArgInt32( other.getArgAsInt32( i ) ) );
		else if ( argType == TYPE_FLOAT )
			args.push_back( new ArgFloat( other.getArgAsFloat( i ) ) );
		else if ( argType == TYPE_STRING )
			args.push_back( new ArgString( other.getArgAsString( i ) ) );
		else
		{
			throw OscExcInvalidArgumentType();
		}
	}
	
	return *this;
}
