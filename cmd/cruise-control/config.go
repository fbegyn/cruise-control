package main

import (
)

// Parse the handles into useable uint32 formates. Returns a map that can be used
// to look up the handle of any object. Each handle parser has the same behavior, but
// needs to be create for each config type (no generics)
func parseHandles(conf Config) (map[string]uint32, error) {
	h1, err := parseQdiscHandle(conf.Qdiscs)
	if err != nil {
		return nil, err
	}
	h2, err := parseClassHandle(conf.Classes)
	if err != nil {
		return nil, err
	}
	h3, err := parseFilterHandle(conf.Filters)
	if err != nil {
		return nil, err
	}

	for k, v := range h2 {
		h1[k] = v
	}
	for k, v := range h3 {
		h1[k] = v
	}
	return h1, nil
}

// Parse parents of a config into a map. This map can be used to lookup the
// parent of any object
func parseParents(handleMap map[string]uint32, conf Config) (map[string]uint32, error) {
	h1, err := parseQdiscParents(handleMap, conf.Qdiscs)
	if err != nil {
		return nil, err
	}
	h2, err := parseClassParents(handleMap, conf.Classes)
	if err != nil {
		return nil, err
	}
	h3, err := parseFilterParents(handleMap, conf.Filters)
	if err != nil {
		return nil, err
	}

	for k, v := range h2 {
		h1[k] = v
	}
	for k, v := range h3 {
		h1[k] = v
	}
	return h1, nil
}

// looks up the handle of an object and returns the uint32 forms
func lookupObjectHandle(handleMap map[string]uint32, object string) (handle uint32, found bool) {
	if h, present := handleMap[object]; present {
		handle = h
	} else {
		return 0, false
	}
	found = true
	return
}

// looks up parent of an object and returns the uint32 forms
func lookupObjectParent(parentMap map[string]uint32, object string) (parent uint32, found bool) {
	if p, present := parentMap[object]; present {
		parent = p
	} else {
		return 0, false
	}
	found = true
	return
}
// looks up the handle and parent of an object and returns the uint32 forms
func lookupObjectHandleParent(handleMap, parentMap map[string]uint32, object string) (handle, parent uint32, found bool) {
	handle, found = lookupObjectHandle(handleMap, object)
	if !found {
		return 0, 0, false
	}
	parent, found = lookupObjectParent(parentMap, object)
	if !found {
		return 0, 0, false
	}
	return
}
