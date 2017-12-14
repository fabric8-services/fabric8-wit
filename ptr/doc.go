// Package ptr provides functions to create on the fly pointer values for some
// built-in types. This might seem stupid but in our codebase we have a lot of
// places in which we just need to create variable just to get the pointer to
// it. That clutters the code a lot and causes distraction. Most of the time
// those strings are just throw-away constants and the pointer "is not really
// important later".
package ptr
