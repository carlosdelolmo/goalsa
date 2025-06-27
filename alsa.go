// Copyright 2015-2016 Cocoon Labs Ltd.
//
// See LICENSE file for terms and conditions.

// Package alsa provides Go bindings to the ALSA library.
package alsa

import (
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"unsafe"
)

/*
#cgo pkg-config: alsa
#include "reader_thread.h"
#include <alsa/asoundlib.h>
#include <stdint.h>
*/
import "C"

// Format is the type used for specifying sample formats.
type Format C.snd_pcm_format_t

// The range of sample formats supported by ALSA.
const (
	FormatS8        = C.SND_PCM_FORMAT_S8
	FormatU8        = C.SND_PCM_FORMAT_U8
	FormatS16LE     = C.SND_PCM_FORMAT_S16_LE
	FormatS16BE     = C.SND_PCM_FORMAT_S16_BE
	FormatU16LE     = C.SND_PCM_FORMAT_U16_LE
	FormatU16BE     = C.SND_PCM_FORMAT_U16_BE
	FormatS24LE     = C.SND_PCM_FORMAT_S24_LE
	FormatS24BE     = C.SND_PCM_FORMAT_S24_BE
	FormatU24LE     = C.SND_PCM_FORMAT_U24_LE
	FormatU24BE     = C.SND_PCM_FORMAT_U24_BE
	FormatS32LE     = C.SND_PCM_FORMAT_S32_LE
	FormatS32BE     = C.SND_PCM_FORMAT_S32_BE
	FormatU32LE     = C.SND_PCM_FORMAT_U32_LE
	FormatU32BE     = C.SND_PCM_FORMAT_U32_BE
	FormatFloatLE   = C.SND_PCM_FORMAT_FLOAT_LE
	FormatFloatBE   = C.SND_PCM_FORMAT_FLOAT_BE
	FormatFloat64LE = C.SND_PCM_FORMAT_FLOAT64_LE
	FormatFloat64BE = C.SND_PCM_FORMAT_FLOAT64_BE
)

var (
	// ErrOverrun signals an overrun error
	ErrOverrun = errors.New("overrun")
	// ErrUnderrun signals an underrun error
	ErrUnderrun = errors.New("underrun")
)

// BufferParams specifies the buffer parameters of a device.
// You do not need to specify all the fields, if you set the BufferParams to 0, default values are used
type BufferParams struct {
	BufferFrames int
	PeriodFrames int
	Periods      int
}

type device struct {
	h            *C.snd_pcm_t
	Channels     int
	Format       Format
	Rate         int
	BufferParams BufferParams
	frames       int
	readerThread *C.reader_thread_state
}

func createError(errorMsg string, errorCode C.int) (err error) {
	strError := C.GoString(C.snd_strerror(errorCode))
	err = fmt.Errorf("%s: %s", errorMsg, strError)
	return
}

func (d *device) createDevice(deviceName string, channels int, format Format, rate int, playback bool, bufferParams BufferParams) (err error) {
	deviceCString := C.CString(deviceName)
	defer C.free(unsafe.Pointer(deviceCString))
	var ret C.int
	if playback {
		ret = C.snd_pcm_open(&d.h, deviceCString, C.SND_PCM_STREAM_PLAYBACK, 0)
	} else {
		ret = C.snd_pcm_open(&d.h, deviceCString, C.SND_PCM_STREAM_CAPTURE, 0)
	}
	if ret < 0 {
		return createError("could not open ALSA device", ret)
	}
	runtime.SetFinalizer(d, (*device).Close)
	var hwParams *C.snd_pcm_hw_params_t
	ret = C.snd_pcm_hw_params_malloc(&hwParams)
	if ret < 0 {
		return createError("could not alloc hw params", ret)
	}
	defer C.snd_pcm_hw_params_free(hwParams)
	ret = C.snd_pcm_hw_params_any(d.h, hwParams)
	if ret < 0 {
		return createError("could not set default hw params", ret)
	}
	ret = C.snd_pcm_hw_params_set_access(d.h, hwParams, C.SND_PCM_ACCESS_RW_INTERLEAVED)
	if ret < 0 {
		return createError("could not set access params", ret)
	}
	ret = C.snd_pcm_hw_params_set_format(d.h, hwParams, C.snd_pcm_format_t(format))
	if ret < 0 {
		return createError("could not set format params", ret)
	}
	ret = C.snd_pcm_hw_params_set_channels(d.h, hwParams, C.uint(channels))
	if ret < 0 {
		return createError("could not set channels params", ret)
	}
	ret = C.snd_pcm_hw_params_set_rate(d.h, hwParams, C.uint(rate), 0)
	if ret < 0 {
		return createError("could not set rate params", ret)
	}

	/*
		// set the buffer time
		buffer_time := C.uint(buffertime)
		ret = C.snd_pcm_hw_params_set_buffer_time_near(d.h, hwParams, &buffer_time, nil)
		if ret < 0 {
			return createError("could not set buffer time", ret)
		}
		bufferSize := C.snd_pcm_uframes_t(0)
		ret = C.snd_pcm_hw_params_get_buffer_size(hwParams, &bufferSize)
		if ret < 0 {
			return createError("could not get buffer size", ret)
		}

		// set the period time
		period_time := C.uint(buffertime / 4)
		ret = C.snd_pcm_hw_params_set_period_time_near(d.h, hwParams, &period_time, nil)
		if ret < 0 {
			return createError("could not set period time", ret)
		}
		periodFrames := C.snd_pcm_uframes_t(0)

		ret = C.snd_pcm_hw_params_get_period_size(hwParams, &periodFrames, nil)
		if ret < 0 {
			return createError("could not get period size", ret)
		}*/

	/*
		// set bufferSize
		var bufferSize = C.snd_pcm_uframes_t(bufferParams.BufferFrames)
		if bufferParams.BufferFrames == 0 {
			// Default buffer size: max buffer size
			ret = C.snd_pcm_hw_params_get_buffer_size_max(hwParams, &bufferSize)
			if ret < 0 {
				return createError("could not get buffer size", ret)
			}
		}
		ret = C.snd_pcm_hw_params_set_buffer_size_near(d.h, hwParams, &bufferSize)
		if ret < 0 {
			return createError("could not set buffer size", ret)
		}
		// Default period size: 1/8 of a second
		var periodFrames = C.snd_pcm_uframes_t(rate / 8)
		if bufferParams.PeriodFrames > 0 {
			periodFrames = C.snd_pcm_uframes_t(bufferParams.PeriodFrames)
		} else if bufferParams.Periods > 0 {
			periodFrames = C.snd_pcm_uframes_t(int(bufferSize) / bufferParams.Periods)
		}

		ret = C.snd_pcm_hw_params_set_period_size_near(d.h, hwParams, &periodFrames, nil)
		if ret < 0 {
			return createError("could not set period size", ret)
		}

		if bufferParams.Periods == 0 {
			bufferParams.Periods = int(bufferSize) / int(periodFrames)
		}*/

	// set buffer size
	var bufferSize = C.snd_pcm_uframes_t(bufferParams.BufferFrames)
	if bufferParams.BufferFrames == 0 {
		// Default buffer size: max buffer size
		ret = C.snd_pcm_hw_params_get_buffer_size_max(hwParams, &bufferSize)
		if ret < 0 {
			return createError("could not get buffer size", ret)
		}
	}
	ret = C.snd_pcm_hw_params_set_buffer_size_near(d.h, hwParams, &bufferSize)
	if ret < 0 {
		return createError("could not set buffer size", ret)
	}

	// set period size
	var periodFrames = C.snd_pcm_uframes_t(bufferParams.PeriodFrames)
	ret = C.snd_pcm_hw_params_set_period_size_near(d.h, hwParams, &periodFrames, nil)
	if ret < 0 {
		return createError("could not set period size", ret)
	}

	// set period number
	var periods = C.uint(bufferParams.Periods)
	ret = C.snd_pcm_hw_params_set_periods_near(d.h, hwParams, &periods, nil)
	if ret < 0 {
		return createError("could not set periods near", ret)
	}

	ret = C.snd_pcm_hw_params_get_periods(hwParams, &periods, nil)
	if ret < 0 {
		return createError("could not get periods", ret)
	}
	ret = C.snd_pcm_hw_params(d.h, hwParams)
	if ret < 0 {
		return createError("could not set hw params", ret)
	}
	d.frames = int(periodFrames)
	d.Channels = channels
	d.Format = format
	d.Rate = rate
	d.BufferParams.BufferFrames = int(bufferSize)
	d.BufferParams.PeriodFrames = int(periodFrames)
	d.BufferParams.Periods = int(periods)
	return
}

// Close closes a device and frees the resources associated with it.
func (d *device) Close() {
	if d.readerThread != nil {
		C.reader_thread_stop(d.readerThread)
		d.readerThread = nil
	}
	if d.h != nil {
		C.snd_pcm_drop(d.h)
		C.snd_pcm_close(d.h)
		d.h = nil
	}
	runtime.SetFinalizer(d, nil)
}

func FormatSampleSize(f Format) (s int) {
	switch f {
	case FormatS8, FormatU8:
		return 1
	case FormatS16LE, FormatS16BE, FormatU16LE, FormatU16BE:
		return 2
	case FormatS24LE, FormatS24BE, FormatU24LE, FormatU24BE, FormatS32LE, FormatS32BE, FormatU32LE, FormatU32BE, FormatFloatLE, FormatFloatBE:
		return 4
	case FormatFloat64LE, FormatFloat64BE:
		return 8
	}
	panic("unsupported format")
}

// CaptureDevice is an ALSA device configured to record audio.
type CaptureDevice struct {
	device
}

// NewCaptureDevice creates a new CaptureDevice object.
func NewCaptureDevice(deviceName string, channels int, format Format, rate int, bufferParams BufferParams) (c *CaptureDevice, err error) {
	c = new(CaptureDevice)
	err = c.createDevice(deviceName, channels, format, rate, false, bufferParams)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *CaptureDevice) StartReadThread() error {
	if c.readerThread != nil {
		return errors.New("Reader thread already running")
	}
	periodBytes := C.int(FormatSampleSize(c.Format) * c.Channels * c.BufferParams.PeriodFrames)
	// Alocate a 1 second buffer
	nbuf := C.int(c.Rate / c.BufferParams.PeriodFrames)
	c.readerThread = C.reader_thread_start(c.h, periodBytes, C.int(c.BufferParams.PeriodFrames), nbuf)
	if c.readerThread == nil {
		return fmt.Errorf("C.reader_thread_start: %s", C.GoString(C.reader_thread_error))
	}
	return nil
}

// Read reads samples into a buffer and returns the amount read.
func (c *CaptureDevice) Read(buffer interface{}) (samples int, err error) {
	bufferType := reflect.TypeOf(buffer)
	if !(bufferType.Kind() == reflect.Array ||
		bufferType.Kind() == reflect.Slice) {
		return 0, errors.New("Read requires an array type")
	}

	sizeError := errors.New("Read requires a matching sample size")
	switch bufferType.Elem().Kind() {
	case reflect.Int8:
		if FormatSampleSize(c.Format) != 1 {
			return 0, sizeError
		}
	case reflect.Int16:
		if FormatSampleSize(c.Format) != 2 {
			return 0, sizeError
		}
	case reflect.Int32, reflect.Float32:
		if FormatSampleSize(c.Format) != 4 {
			return 0, sizeError
		}
	case reflect.Float64:
		if FormatSampleSize(c.Format) != 8 {
			return 0, sizeError
		}
	default:
		return 0, errors.New("Read does not support this format")
	}

	val := reflect.ValueOf(buffer)
	length := val.Len()
	sliceData := val.Slice(0, length)

	frames := length / c.Channels
	bufPtr := unsafe.Pointer(sliceData.Index(0).Addr().Pointer())

	if c.readerThread != nil {
		if frames != c.BufferParams.PeriodFrames {
			return 0, errors.New("buffer size must match period")
		}
		rc := C.reader_thread_poll(c.readerThread, bufPtr)
		if rc == 1 {
			return 0, ErrOverrun
		} else if rc != 0 {
			return 0, fmt.Errorf("read error: %s", C.GoString(C.reader_thread_error))
		}
		samples = frames * c.Channels
	} else {
		ret := C.snd_pcm_readi(c.h, bufPtr, C.snd_pcm_uframes_t(frames))

		if ret == -C.EPIPE {
			C.snd_pcm_prepare(c.h)
			return 0, ErrOverrun
		} else if ret < 0 {
			return 0, createError("read error", C.int(ret))
		}
		samples = int(ret) * c.Channels
	}
	return
}

// PlaybackDevice is an ALSA device configured to playback audio.
type PlaybackDevice struct {
	device
}

// NewPlaybackDevice creates a new PlaybackDevice object.
func NewPlaybackDevice(deviceName string, channels int, format Format, rate int, bufferParams BufferParams) (p *PlaybackDevice, err error) {
	p = new(PlaybackDevice)
	err = p.createDevice(deviceName, channels, format, rate, true, bufferParams)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// Write writes a buffer of data to a playback device.
func (p *PlaybackDevice) Write(buffer interface{}) (samples int, err error) {
	bufferType := reflect.TypeOf(buffer)
	if !(bufferType.Kind() == reflect.Array ||
		bufferType.Kind() == reflect.Slice) {
		return 0, errors.New("Write requires an array type")
	}

	sizeError := errors.New("Write requires a matching sample size")
	switch bufferType.Elem().Kind() {
	case reflect.Int8:
		if FormatSampleSize(p.Format) != 1 {
			return 0, sizeError
		}
	case reflect.Int16:
		if FormatSampleSize(p.Format) != 2 {
			return 0, sizeError
		}
	case reflect.Int32, reflect.Float32:
		if FormatSampleSize(p.Format) != 4 {
			return 0, sizeError
		}
	case reflect.Float64:
		if FormatSampleSize(p.Format) != 8 {
			return 0, sizeError
		}
	default:
		return 0, errors.New("Write does not support this format")
	}

	val := reflect.ValueOf(buffer)
	length := val.Len()
	sliceData := val.Slice(0, length)

	var frames = C.snd_pcm_uframes_t(length / p.Channels)
	bufPtr := unsafe.Pointer(sliceData.Index(0).Addr().Pointer())

	ret := C.snd_pcm_writei(p.h, bufPtr, frames)
	if ret == -C.EPIPE {
		C.snd_pcm_prepare(p.h)
		return 0, ErrUnderrun
	} else if ret < 0 {
		return 0, createError("write error", C.int(ret))
	}
	samples = int(ret) * p.Channels
	return
}

// FormatToString returns a string representation of the format.
func FormatToString(format Format) string {
	switch format {
	case FormatS8:
		return "S8"
	case FormatU8:
		return "U8"
	case FormatS16LE:
		return "S16LE"
	case FormatS16BE:
		return "S16BE"
	case FormatU16LE:
		return "U16LE"
	case FormatU16BE:
		return "U16BE"
	case FormatS24LE:
		return "S24LE"
	case FormatS24BE:
		return "S24BE"
	case FormatU24LE:
		return "U24LE"
	case FormatU24BE:
		return "U24BE"
	case FormatS32LE:
		return "S32LE"
	case FormatS32BE:
		return "S32BE"
	case FormatU32LE:
		return "U32LE"
	case FormatU32BE:
		return "U32BE"
	case FormatFloatLE:
		return "FloatLE"
	case FormatFloatBE:
		return "FloatBE"
	case FormatFloat64LE:
		return "Float64LE"
	case FormatFloat64BE:
		return "Float64BE"
	default:
		return "Unknown Format"
	}
}

// FormatToString returns the format from a string representation.
func StringToFormat(formatStr string) Format {
	switch formatStr {
	case "S8":
		return FormatS8
	case "U8":
		return FormatU8
	case "S16LE":
		return FormatS16LE
	case "S16BE":
		return FormatS16BE
	case "U16LE":
		return FormatU16LE
	case "U16BE":
		return FormatU16BE
	case "S24LE":
		return FormatS24LE
	case "S24BE":
		return FormatS24BE
	case "U24LE":
		return FormatU24LE
	case "U24BE":
		return FormatU24BE
	case "S32LE":
		return FormatS32LE
	case "S32BE":
		return FormatS32BE
	case "U32LE":
		return FormatU32LE
	case "U32BE":
		return FormatU32BE
	case "FloatLE":
		return FormatFloatLE
	case "FloatBE":
		return FormatFloatBE
	case "Float64LE":
		return FormatFloat64LE
	case "Float64BE":
		return FormatFloat64BE
	default:
		return -1 // Unknown format
	}
}
