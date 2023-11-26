//go:build windows
// +build windows

package kit

// #cgo LDFLAGS: -L. "${SRCDIR}/../SenselLib/x64/LibSensel.dll"
/*
#include <stdlib.h>
#include <stdio.h>
#include "../SenselLib/include/sensel.h"

// THIS STRUCTURE NEEDS TO MATCH THE C VERSION
// EVENTUALLY I SHOULD GET RID OF THIS
typedef struct goSenselSensorInfo
{
    unsigned char   max_contacts;       // Maximum number of contacts the sensor supports
    unsigned short  num_rows;           // Total number of rows
    unsigned short  num_cols;           // Total number of columns
    float           width;              // Width of the sensor in millimeters
    float           height;             // Height of the sensor in millimeters
} goSenselSensorInfo;

// THIS STRUCTURE DOES NOT NEED TO MATCH THE C VERSION
typedef struct goSenselFirmwareInfo
{
    unsigned char  fw_protocol_version; // Sensel communication protocol supported by the device
    unsigned char  fw_version_major;    // Major version of the firmware
    unsigned char  fw_version_minor;    // Minor version of the firmware
    unsigned short fw_version_build;    // ??
    unsigned char  fw_version_release;  // ??
    unsigned short device_id;           // Sensel device type
    unsigned char  device_revision;     // Device revision
} goSenselFirmwareInfo;

// THIS STRUCTURE DOES NOT NEED TO MATCH THE C VERSION
typedef struct goSenselFrameData
{
    unsigned char   content_bit_mask;  // Data contents of the frame
    int             lost_frame_count;  // Number of frames dropped
    unsigned char   n_contacts;        // Number of contacts
} goSenselFrameData;

// THIS STRUCTURE DOES NOT NEED TO MATCH THE C VERSION
typedef struct goSenselContact
{
    // unsigned char        content_bit_mask;   // Mask of what contact data is valid
    unsigned char        id;                 // Contact id
    unsigned int         state;              // Contact state (enum SenselContactState)
    float                x_pos;              // X position in mm
    float                y_pos;              // Y position in mm
    float                total_force;        // Total contact force in grams
    float                area;               // Area in sensor elements
} goSenselContact;

typedef struct OneMorph {
	// void*			handle;
	SENSEL_HANDLE		handle;
	SenselFrameData *frame;
} OneMorph;

// The order in this list is the idx value
OneMorph Morphs[SENSEL_MAX_DEVICES];

int senselLoaded = 0;
SenselDeviceList sensellist;

void
SenselInit()
{
	if ( ! senselLoaded ) {
		senselGetDeviceList(&sensellist);
		senselLoaded = 1;
	}
}

int
SenselNumDevices()
{
	SenselInit();
	return sensellist.num_devices;
}

char *
SenselDeviceSerialNum(unsigned char idx) {
	if ( idx < 0 || idx >= SENSEL_MAX_DEVICES ) {
		return "InvalidDeviceIndex";
	}
	return sensellist.devices[idx].serial_num;
}

// int
// SenselOpenDeviceBySerialNum(void** handle, char* serial_num)
// {
// 	return senselOpenDeviceBySerialNum(handle,serial_num);
// }

int
SenselOpenDeviceByID(unsigned char idx)
{
	if ( idx < 0 || idx >= SENSEL_MAX_DEVICES ) {
		return SENSEL_ERROR;
	}
	SENSEL_HANDLE h;
	SenselStatus status = senselOpenDeviceByID(&h,idx);
	Morphs[idx].handle = h;
	return status;
}

int
SenselGetSensorInfo(unsigned char idx, goSenselSensorInfo *info)
{
	if ( idx < 0 || idx >= SENSEL_MAX_DEVICES ) {
		return SENSEL_ERROR;
	}
	SENSEL_HANDLE handle = Morphs[idx].handle;
	SenselSensorInfo *senselinfo = (SenselSensorInfo*)info;
	return senselGetSensorInfo(handle,senselinfo);
}

int
SenselGetFirmwareInfo(unsigned char idx, goSenselFirmwareInfo *goinfo)
{
	if ( idx < 0 || idx >= SENSEL_MAX_DEVICES ) {
		return SENSEL_ERROR;
	}
	SENSEL_HANDLE handle = Morphs[idx].handle;
	SenselFirmwareInfo info;
	SenselStatus s = senselGetFirmwareInfo(handle,&info);
	if ( s == SENSEL_OK ) {
		goinfo->fw_protocol_version = info.fw_protocol_version;
		goinfo->fw_version_major = info.fw_version_major;
		goinfo->fw_version_minor = info.fw_version_minor;
		goinfo->fw_version_build = info.fw_version_build;
		goinfo->fw_version_release = info.fw_version_release;
		goinfo->device_id = info.device_id;
		goinfo->device_revision = info.device_revision;
	}
	return s;
}

int
SenselSetupAndStart(unsigned char idx)
{
	if ( idx < 0 || idx >= SENSEL_MAX_DEVICES ) {
		return -1;
	}
	SENSEL_HANDLE handle = Morphs[idx].handle;

	unsigned char v[1] = {255};
	SenselStatus s;
	s = senselWriteReg(handle,0xD0,1,v);  // This disables timeouts
	if ( s != SENSEL_OK ) {
		return -2;
	}

	s = senselSetFrameContent(handle, FRAME_CONTENT_CONTACTS_MASK);
	if ( s != SENSEL_OK ) {
		return -3;
	}

	s = senselAllocateFrameData(handle, &(Morphs[idx].frame));
	if ( s != SENSEL_OK ) {
		return -4;
	}

	s = senselStartScanning(handle);
	if ( s != SENSEL_OK ) {
		return -5;
	}

	for (int led = 0; led < 16; led++) {
		s = senselSetLEDBrightness(handle, led, 0); //turn off LED
		if ( s != SENSEL_OK ) {
			return -6;
		}
	}
	return SENSEL_OK;
}

int
SenselReadSensor(unsigned char idx)
{
	if ( idx < 0 || idx >= SENSEL_MAX_DEVICES ) {
		return SENSEL_ERROR;
	}
	SENSEL_HANDLE handle = Morphs[idx].handle;
	SenselStatus s = senselReadSensor(handle);
	return s;
}

int
SenselGetNumAvailableFrames(unsigned char idx)
{
	if ( idx < 0 || idx >= SENSEL_MAX_DEVICES ) {
		return SENSEL_ERROR;
	}
	SENSEL_HANDLE handle = Morphs[idx].handle;
	unsigned int nframes;
	SenselStatus s = senselGetNumAvailableFrames(handle,&nframes);
	if ( s != SENSEL_OK ) {
		return -1;
	}
	return nframes;
}

int
SenselGetFrame(unsigned char idx, goSenselFrameData *goFrame)
{
	if ( idx < 0 || idx >= SENSEL_MAX_DEVICES ) {
		return SENSEL_ERROR;
	}
	SENSEL_HANDLE handle = Morphs[idx].handle;
	SenselStatus s = senselGetFrame(handle,Morphs[idx].frame);
	SenselFrameData *f = Morphs[idx].frame;
	goFrame->n_contacts = f->n_contacts;
    goFrame->lost_frame_count = f->lost_frame_count;
    goFrame->content_bit_mask = f->content_bit_mask;
	return s;
}

int
SenselGetContact(unsigned char idx, unsigned char contactNum, goSenselContact *goContact)
{
	if ( idx < 0 || idx >= SENSEL_MAX_DEVICES ) {
		return SENSEL_ERROR;
	}
	SenselFrameData *f = Morphs[idx].frame;
	if ( contactNum >= f->n_contacts ) {
		return SENSEL_ERROR;
	}
	goContact->id = f->contacts[contactNum].id;
	goContact->x_pos = f->contacts[contactNum].x_pos;
	goContact->y_pos = f->contacts[contactNum].y_pos;
	goContact->state = f->contacts[contactNum].state;
	goContact->total_force = f->contacts[contactNum].total_force;
	goContact->area = f->contacts[contactNum].area;
	return SENSEL_OK;
}


*/
import "C"

import (
	"fmt"
	"time"
)

type oneMorph struct {
	idx              uint8
	opened           bool
	serialNum        string
	width            float32
	height           float32
	fwVersionMajor   uint8
	fwVersionMinor   uint8
	fwVersionBuild   uint8
	fwVersionRelease uint8
	deviceID         int
	morphtype        string // "corners", "quadrants"
	currentTag       string // "A", "B", "C", "D" - it can change dynamically
	previousTag      string // "A", "B", "C", "D" - it can change dynamically
	contactIdToGid   map[int]int
}

var morphMaxForce float32 = 1000.0

var allMorphs []*oneMorph

// StartMorph xxx
func StartMorph(callback CursorCallbackFunc, forceFactor float32) {
	err := WinMorphInitialize()
	if err != nil {
		LogIfError(err)
		return
	}
	if len(allMorphs) == 0 {
		LogInfo("No Morphs were found")
		return
	}
	for {
		for _, m := range allMorphs {
			if m.opened {
				m.readFrames(callback, forceFactor)
			}
		}
		time.Sleep(time.Millisecond)
	}
}

// CursorDown etc match values in sensel.h
const (
	CursorDown = 1
	CursorDrag = 2
	CursorUp   = 3
)

func (m *oneMorph) readFrames(callback CursorCallbackFunc, forceFactor float32) {
	status := C.SenselReadSensor(C.uchar(m.idx))
	if status != C.SENSEL_OK {
		LogWarn("SenselReadSensor for", "idx", m.idx, "status", status)
		LogWarn("Morph has been disabled due to SenselReadSensor errors", "serialnum", m.serialNum)
		m.opened = false
	}
	numFrames := C.SenselGetNumAvailableFrames(C.uchar(m.idx))
	if numFrames <= 0 {
		return
	}
	nf := int(numFrames)
	for n := 0; n < nf; n++ {
		var frame C.struct_goSenselFrameData
		status := C.SenselGetFrame(C.uchar(m.idx), &frame)
		if status != C.SENSEL_OK {
			LogWarn("SenselGetFrame", "idx", m.idx, "status", status)
			continue
		}
		nc := int(frame.n_contacts)
		for n := 0; n < nc; n++ {
			var contact C.struct_goSenselContact
			status = C.SenselGetContact(C.uchar(m.idx), C.uchar(n), &contact)
			if status != C.SENSEL_OK {
				LogWarn("SenselGetContact of morph", "idx", m.idx, "n", n, "status", status)
				continue
			}
			xNorm := float32(contact.x_pos) / m.width
			yNorm := float32(contact.y_pos) / m.height
			zNorm := float32(contact.total_force) / morphMaxForce
			zNorm *= forceFactor
			area := float32(contact.area)
			var ddu string
			switch contact.state {
			case CursorDown:
				ddu = "down"
			case CursorDrag:
				ddu = "drag"
			case CursorUp:
				ddu = "up"
			default:
				LogWarn("Invalid value", "state", contact.state)
				continue
			}

			switch m.morphtype {

			case "corners":

				// If the position is in one of the corners,
				// we change the source to that corner.
				edge := float32(0.075)
				cornerSource := ""
				if xNorm < edge && yNorm < edge {
					cornerSource = "A"
				} else if xNorm < edge && yNorm > (1.0-edge) {
					cornerSource = "B"
				} else if xNorm > (1.0-edge) && yNorm > (1.0-edge) {
					cornerSource = "C"
				} else if xNorm > (1.0-edge) && yNorm < edge {
					cornerSource = "D"
				}
				if cornerSource != m.currentTag {
					LogInfo("Switching corners pad", "source", cornerSource)
					ce := NewCursorClearEvent()
					callback(ce)
					m.currentTag = cornerSource
					continue // loop, doesn't send a cursor event
				}

			case "quadrants":

				// This method splits a single pad into quadrants.
				// Adjust the xNorm and yNorm values to provide
				// full range 0-1 within each quadrant.
				quadSource := ""
				switch {
				case xNorm < 0.5 && yNorm < 0.5:
					quadSource = "A"
				case xNorm < 0.5 && yNorm >= 0.5:
					quadSource = "B"
					yNorm = yNorm - 0.5
				case xNorm >= 0.5 && yNorm >= 0.5:
					quadSource = "C"
					xNorm = xNorm - 0.5
					yNorm = yNorm - 0.5
				case xNorm >= 0.5 && yNorm < 0.5:
					quadSource = "D"
					xNorm = xNorm - 0.5
				default:
					LogWarn("unable to find QUAD source", "x", xNorm, "y", yNorm)
					continue
				}
				xNorm *= 2.0
				yNorm *= 2.0
				m.currentTag = quadSource

			case "A", "B", "C", "D":
				m.currentTag = m.morphtype

			default:
				LogWarn("Unknown morphtype", "morphtype", m.morphtype)
				m.currentTag = ""
			}

			if m.currentTag == "" {
				LogWarn("Hey! currentTag not set, assuming A")
				m.currentTag = "A"
			}

			contactid := int(contact.id)
			gid, ok := m.contactIdToGid[contactid]
			if !ok {
				// If we've never seen this contact before, create a new cid...
				gid = TheCursorManager.UniqueGid()
				m.contactIdToGid[contactid] = gid
			} else if m.currentTag != m.previousTag {
				// If we're switching to a new source, clear existing cursors...
				ce := NewCursorClearEvent()
				callback(ce)
				// and create a new cid...
				gid = TheCursorManager.UniqueGid()
			}

			m.previousTag = m.currentTag

			LogOfType("morph", "Morph",
				"idx", m.idx,
				"contactid", contactid,
				"gid", gid,
				"n", n,
				"contactstate", contact.state,
				"xNorm", xNorm,
				"yNorm", yNorm,
				"zNorm", zNorm)

			// make the coordinate space match OpenGL and Freeframe
			yNorm = 1.0 - yNorm

			// Make sure we don't send anyting out of bounds
			if yNorm < 0.0 {
				yNorm = 0.0
			} else if yNorm > 1.0 {
				yNorm = 1.0
			}
			if xNorm < 0.0 {
				xNorm = 0.0
			} else if xNorm > 1.0 {
				xNorm = 1.0
			}

			pos := CursorPos{xNorm, yNorm, zNorm}
			ce := NewCursorEvent(gid, m.currentTag, ddu, pos)
			ce.Area = area
			callback(ce)
		}
	}
}

// Initialize xxx
func WinMorphInitialize() error {

	numdevices := int(C.SenselNumDevices())
	allMorphs = make([]*oneMorph, numdevices)

	for idx := uint8(0); idx < uint8(numdevices); idx++ {

		m := &oneMorph{
			contactIdToGid: map[int]int{},
		}
		allMorphs[idx] = m
		m.idx = idx
		m.serialNum = C.GoString(C.SenselDeviceSerialNum(C.uchar(idx)))

		status := C.SenselOpenDeviceByID(C.uchar(idx))
		if status != C.SENSEL_OK {
			return fmt.Errorf("SenselOpenDeviceBySerialNum of idx=%d returned %d", idx, status)
		}

		var sensorinfo C.struct_goSenselSensorInfo
		status = C.SenselGetSensorInfo(C.uchar(idx), &sensorinfo)
		if status != C.SENSEL_OK {
			return fmt.Errorf("SenselGetSensorInfo of idx=%d returned %d", idx, status)
		}

		var firmwareinfo C.struct_goSenselFirmwareInfo
		status = C.SenselGetFirmwareInfo(C.uchar(idx), &firmwareinfo)
		if status != C.SENSEL_OK {
			return fmt.Errorf("SenselGetFirmwareInfo of %s returned %d", m.serialNum, status)
		}

		status = C.SenselSetupAndStart(C.uchar(m.idx))
		if status != C.SENSEL_OK {
			return fmt.Errorf("SenselSetupAndStart of %s returned %d", m.serialNum, status)
		}

		m.opened = true
		m.width = float32(sensorinfo.width)
		m.height = float32(sensorinfo.height)
		m.fwVersionMajor = uint8(firmwareinfo.fw_version_major)
		m.fwVersionMinor = uint8(firmwareinfo.fw_version_minor)
		m.fwVersionBuild = uint8(firmwareinfo.fw_version_build)
		m.fwVersionRelease = uint8(firmwareinfo.fw_version_release)
		m.deviceID = int(firmwareinfo.device_id)

		morphtype, ok := MorphDefs[m.serialNum]
		if !ok {
			// It's not explicitly present in morphs.json
			t, err := GetParam("global.morphtype")
			if err != nil {
				return err
			}
			morphtype = t
			LogInfo("Morph serial# isn't in morphs.json, using engine.morphtype", "serialnum", m.serialNum, "morphtype", morphtype)
		}

		m.morphtype = morphtype
		switch m.morphtype {
		case "corners", "quadrants":
			m.currentTag = "A"
		case "A", "B", "C", "D":
			m.currentTag = morphtype
		default:
			LogWarn("Unexpected morphtype", "morphtype", morphtype)
		}

		// Don't use Debug.Morph, this should always gets logged
		firmware := fmt.Sprintf("%d.%d.%d", m.fwVersionMajor, m.fwVersionMinor, m.fwVersionBuild)
		LogInfo("Morph Opened and Started", "idx", m.idx, "serial", m.serialNum, "firmware", firmware)
	}
	return nil
}
