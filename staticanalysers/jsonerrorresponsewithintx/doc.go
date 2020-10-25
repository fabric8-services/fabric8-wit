// Package jsonerrorresponsewithintx parses Go files and searches for calls to
// jsonapi.JSONErrorResponse. If it finds one it traverses up the call graph to
// that place to see if the call was being made from within a transaction that
// was initiated with application.Transactional.
//
// There was an issue in OpenShift.io once [1] and one of the fixes [2] involved
// changing a lot of places where exactly calls like the ones described above
// needed to be addressed. The jsonerrorresponsewithintx static anylser shall
// help to keep such code from ever being merged into our repo again.
//
// [1]: https://github.com/openshiftio/openshift.io/issues/2689
//
// [2]: https://github.com/openshiftio/openshift.io/issues/2687
package jsonerrorresponsewithintx
