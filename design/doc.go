// Package design contains the generic API machinery code of our adder API generated
// using goa framework. This generated API accepts HTTP GET/POST/PATCH/PUT/DELETE requests
// from multiple clients.
package design

import "strings"

type fieldDesc struct {
	desc         string
	mandOnCreate bool
	mandOnUpdate bool
}

// desc returns a field description and by default the field is marked as
// optional in creation and update of resource.
func desc(str ...string) fieldDesc {
	var tmp string
	if len(str) > 0 {
		tmp = str[0]
	}
	return fieldDesc{
		desc: tmp,
	}
}

func (d fieldDesc) String() string {
	s := d.desc
	if !strings.HasSuffix(strings.TrimSpace(s), ".") {
		s += ". "
	}
	if d.mandOnCreate {
		s += "\n This is MANDATORY on creation of resource."
	} else {
		//s += "\n This is OPTIONAL on creation of resource."
	}
	if d.mandOnUpdate {
		s += "\n This is MANDATORY on update of resource."
	} else {
		//s += "\n This is OPTIONAL on update of resource."
	}
	return s
}

func (d fieldDesc) mandatoryOnCreate(b ...bool) fieldDesc {
	res := d
	if len(b) > 0 {
		res.mandOnCreate = b[0]
	} else {
		res.mandOnCreate = true
	}

	return res
}
func (d fieldDesc) optionalOnCreate(b ...bool) fieldDesc {
	res := d
	if len(b) > 0 {
		res.mandOnCreate = !b[0]
	} else {
		res.mandOnCreate = false
	}
	return res
}
func (d fieldDesc) mandatoryOnUpdate(b ...bool) fieldDesc {
	res := d
	if len(b) > 0 {
		res.mandOnUpdate = b[0]
	} else {
		res.mandOnUpdate = true
	}
	return res
}
func (d fieldDesc) optionalOnUpdate(b ...bool) fieldDesc {
	res := d
	if len(b) > 0 {
		res.mandOnUpdate = !b[0]
	} else {
		res.mandOnUpdate = false
	}
	return res
}
