#pragma once

#include <exception>
#include <string>
#include <windows.h>
#include "NosuchUtil.h"

class NosuchException : public std::exception {
};
class NosuchBadValueException : public NosuchException {
};
class NosuchUnableToLoadException : public NosuchException {
};
class NosuchMissingItemException : public NosuchException {
};
class NosuchMissingValueException : public NosuchException {
};
class NosuchNoParametersException : public NosuchException {
};
class NosuchUnexpectedTypeException : public NosuchException {
};
class NosuchNotEnoughArgumentsException : public NosuchException {
};
class NosuchBadTypeOfArgumentException : public NosuchException {
};
class NosuchArrayIsEmptyException : public NosuchException {
};
class NosuchUnrecognizedTypeException : public NosuchException {
};
class NosuchMiscException : public NosuchException {
};
