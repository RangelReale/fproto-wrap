package fproto_gowrap

import (
	"fmt"
	"sort"
	"strings"
)

// Helper to facilitate building struct tags
type StructTag struct {
	Tags   map[string]string
	Append string
}

func NewStructTag() *StructTag {
	return &StructTag{
		Tags: make(map[string]string),
	}
}

func (s *StructTag) Output() string {
	var ret []string

	// loop tags in ascending order
	keys := make([]string, 0)
	for k, _ := range s.Tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, tn := range keys {
		ret = append(ret, fmt.Sprintf("%s:\"%s\"", tn, s.Tags[tn]))
	}
	if s.Append != "" {
		ret = append(ret, s.Append)
	}
	if len(ret) == 0 {
		return ""
	}

	return "`" + strings.Join(ret, " ") + "`"
}

func (s *StructTag) OutputWithSpace() string {
	ret := s.Output()
	if ret != "" {
		return " " + ret
	}
	return ret
}

func (s *StructTag) Set(name, value string) {
	s.Tags[name] = value
}

func (s *StructTag) Get(name string) (value string, ok bool) {
	value, ok = s.Tags[name]
	return
}

func (s *StructTag) Clear() {
	s.Tags = make(map[string]string)
}

func (s *StructTag) Delete(name string) {
	delete(s.Tags, name)
}

func (s *StructTag) GetAppend() string {
	return s.Append
}

func (s *StructTag) SetAppend(append string) {
	s.Append = append
}
