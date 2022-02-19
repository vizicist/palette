/*
 * Produced by:
 *
 * Derivative Inc
 *		401 Richmond Street West, Unit 386
 *		Toronto, Ontario
 *		Canada   M5V 3A8
 *		416-591-3555
 *
 * NAME:		CHOP_SharedMemHeader.h ( Library, C++)
 *
 * COMMENTS:
 */

#ifndef __CHOP_SharedMemHeader__
#define __CHOP_SharedMemHeader__

#define CHOP_SHM_MAGIC_NUMBER 		0x12ED54CF
#define CHOP_SHM_VERSION		1

class CHOP_SharedMemHeader
{
public:

    // Must be set to CHOP_SHM_MAGIC_NUMBER
    int magicNumber;

    // Must be set to CHOP_SHM_VERSION
    int version;

    // Specifies the size of the shared memory buffer, including the header
    int size;

    // The number of channels that are being transmited
    int numChans;

    // The length of the channels that are being transmited (number of samples)
    int length;

    // The sample rate of the data thats being transmited
    float sampleRate;

    // Set this to 1 if the shared memory data starts with channel
    // names, followed by the channel data
    // Set this to 0 if the shared memory data just starts with the
    // channel data and has no channel names.
    int namesSent;

    // Offset in bytes from the start of the buffer to where the channel data
    // is.
    // This value must always be a multiple of 4, for byte alignment when
    // reading and writting floats.
    // You still typically place your channel data directly after the header
    // in which case you would set this to sizeof(CHOP_SharedMemHeader)
    int channelDataOffset;

    // Offset in byte from the start of the buffer to where
    // the channel names are
    // Typically you would place your channel names after the channel data,
    // so this would be set to 
    // channelDataOffset + (numChans * length * sizeof(float))
    // This value is ignored if namesSent == 0
    int nameDataOffset;


    float		*getChannelData()
    {
		char *bptr = (char*)this;
		return (float*)(bptr + channelDataOffset);
    }

    char		*getNameData()
    {
		char *bptr = (char*)this;
		return bptr + nameDataOffset;
    }

    // This function does two things, it will shift the pointer location
    // forward up to 3 bytes to the nearest 4 byte boundary, for byte
    // alignment when reading/writting floats.
    // It will also return the number of bytes it had to shift the pointer
    // forward to align it
    // You typically don't need to use this function
    static int alignPointer(void *&ptr)
    {
		unsigned int pad = (unsigned int)ptr % sizeof(float);
		if (pad > 0)
		{
		    int align = (sizeof(float) - pad);
		    ptr = (char*)ptr + align;
		    return align;
		}
		else
		{
		    // No alignment needed
		    return 0;
		}
    }


};

#endif
