package codebase

import "errors"

// CodebaseContent defines all parameters those are useful to associate Che Editor's window to a WI
type CodebaseContent struct {
	Repository string `json:"repository"`
	Branch     string `json:"branch"`
	FileName   string `json:"filename"`
	LineNumber int    `json:"linenumber"`
}

// Following keys define attribute names in the map of Codebase
const (
	RepositoryKey = "repository"
	BranchKey     = "branch"
	FileNameKey   = "filename"
	LineNumberKey = "linenumber"
)

// ToMap converts CodebaseContent to a map of string->Interface{}
func (c *CodebaseContent) ToMap() map[string]interface{} {
	res := make(map[string]interface{})
	res[RepositoryKey] = c.Repository
	res[BranchKey] = c.Branch
	res[FileNameKey] = c.FileName
	res[LineNumberKey] = c.LineNumber
	return res
}

// NewCodebase build CodebaseContent instance from input Map.
func NewCodebase(value map[string]interface{}) (CodebaseContent, error) {
	cb := CodebaseContent{}
	validKeys := []string{RepositoryKey, BranchKey, FileNameKey, LineNumberKey}
	for _, key := range validKeys {
		if v, ok := value[key]; ok {
			switch key {
			case RepositoryKey:
				cb.Repository = v.(string)
			case BranchKey:
				cb.Branch = v.(string)
			case FileNameKey:
				cb.FileName = v.(string)
			case LineNumberKey:
				switch v.(type) {
				case int:
					cb.LineNumber = v.(int)
				case float64:
					y := v.(float64)
					cb.LineNumber = int(y)
				}
			}
		}
	}
	emptyCodebase := CodebaseContent{}
	if cb == emptyCodebase {
		// Not a single valid key found in `value`
		return emptyCodebase, errors.New("Invalid keys for Codebase")
	}
	return cb, nil
}
