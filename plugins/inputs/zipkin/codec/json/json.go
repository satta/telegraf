package json

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/influxdata/telegraf/plugins/inputs/zipkin/codec"
)

// JSON decodes spans from  bodies `POST`ed to the spans endpoint
type JSON struct{}

// Decode unmarshals and validates the JSON body
func (j *JSON) Decode(octets []byte) ([]codec.Span, error) {
	var spans []span
	err := json.Unmarshal(octets, &spans)
	if err != nil {
		return nil, err
	}
	res := make([]codec.Span, len(spans))
	for i, s := range spans {
		res[i] = &s
	}
	return res, nil
}

type span struct {
	TraceID  string             `json:"traceId"`
	SpanName string             `json:"name"`
	ParentID string             `json:"parentId,omitempty"`
	ID       string             `json:"id"`
	Time     *int64             `json:"timestamp,omitempty"`
	Dur      *int64             `json:"duration,omitempty"`
	Debug    bool               `json:"debug,omitempty"`
	Anno     []annotation       `json:"annotations"`
	BAnno    []binaryAnnotation `json:"binaryAnnotations"`
}

func (s *span) Trace() (string, error) {
	return TraceIDFromString(s.TraceID)
}

func (s *span) SpanID() (string, error) {
	return IDFromString(s.ID)
}

func (s *span) Parent() (string, error) {
	if s.ParentID == "" {
		return "", nil
	}
	return IDFromString(s.ParentID)
}

func (s *span) Name() string {
	return s.SpanName
}

func (s *span) Annotations() []codec.Annotation {
	res := make([]codec.Annotation, len(s.Anno))
	for i, a := range s.Anno {
		res[i] = &a
	}
	return res
}

func (s *span) BinaryAnnotations() []codec.BinaryAnnotation {
	res := make([]codec.BinaryAnnotation, len(s.BAnno))
	for i, a := range s.BAnno {
		res[i] = &a
	}
	return res
}

func (s *span) Timestamp() time.Time {
	if s.Time == nil {
		return time.Time{}
	}
	return codec.MicroToTime(*s.Time)
}

func (s *span) Duration() time.Duration {
	if s.Dur == nil {
		return 0
	}
	return time.Duration(*s.Dur) * time.Microsecond
}

type annotation struct {
	Endpoint *endpoint `json:"endpoint,omitempty"`
	Time     int64     `json:"timestamp"`
	Val      string    `json:"value,omitempty"`
}

func (a *annotation) Timestamp() time.Time {
	return codec.MicroToTime(a.Time)
}

func (a *annotation) Value() string {
	return a.Val
}

func (a *annotation) Host() codec.Endpoint {
	return a.Endpoint
}

type binaryAnnotation struct {
	K        string    `json:"key"`
	V        string    `json:"value"`
	Endpoint *endpoint `json:"endpoint,omitempty"`
}

func (b *binaryAnnotation) Key() string {
	return b.K
}

func (b *binaryAnnotation) Value() string {
	return b.V
}

func (b *binaryAnnotation) Host() codec.Endpoint {
	return b.Endpoint
}

type endpoint struct {
	ServiceName string `json:"serviceName"`
	Ipv4        string `json:"ipv4"`
	Ipv6        string `json:"ipv6,omitempty"`
	Port        int    `json:"port"`
}

func (e *endpoint) Host() string {
	if e.Port != 0 {
		return fmt.Sprintf("%s:%d", e.Ipv4, e.Port)
	}
	return e.Ipv4
}

func (e *endpoint) Name() string {
	return e.ServiceName
}

// TraceIDFromString creates a TraceID from a hexadecimal string
func TraceIDFromString(s string) (string, error) {
	var hi, lo uint64
	var err error
	if len(s) > 32 {
		return "", fmt.Errorf("TraceID cannot be longer than 32 hex characters: %s", s)
	} else if len(s) > 16 {
		hiLen := len(s) - 16
		if hi, err = strconv.ParseUint(s[0:hiLen], 16, 64); err != nil {
			return "", err
		}
		if lo, err = strconv.ParseUint(s[hiLen:], 16, 64); err != nil {
			return "", err
		}
	} else {
		if lo, err = strconv.ParseUint(s, 16, 64); err != nil {
			return "", err
		}
	}
	if hi == 0 {
		return fmt.Sprintf("%x", lo), nil
	}
	return fmt.Sprintf("%x%016x", hi, lo), nil
}

// IDFromString creates a decimal id from a hexadecimal string
func IDFromString(s string) (string, error) {
	if len(s) > 16 {
		return "", fmt.Errorf("ID cannot be longer than 16 hex characters: %s", s)
	}
	id, err := strconv.ParseUint(s, 16, 64)
	if err != nil {
		return "", err
	}
	return strconv.FormatUint(id, 10), nil
}
