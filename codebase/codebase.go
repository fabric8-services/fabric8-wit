package codebase

import "github.com/almighty/almighty-core/errors"

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

// IsValid perform following checks
// Repository value is mandatory
func (c *CodebaseContent) IsValid() error {
	if c.Repository == "" {
		return errors.NewBadParameterError("system.codebase", RepositoryKey+" is mandatory")
	}
	return nil
}

// NewCodebaseContent builds CodebaseContent instance from input Map.
func NewCodebaseContent(value map[string]interface{}) (CodebaseContent, error) {
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
	err := cb.IsValid()
	if err != nil {
		return cb, err
	}
	return cb, nil
}

// NewCodebaseContentFromValue builds CodebaseContent from interface{}
func NewCodebaseContentFromValue(value interface{}) (*CodebaseContent, error) {
	if value == nil {
		return nil, nil
	}
	switch value.(type) {
	case CodebaseContent:
		result := value.(CodebaseContent)
		return &result, nil
	case map[string]interface{}:
		result, err := NewCodebaseContent(value.(map[string]interface{}))
		if err != nil {
			return nil, err
		}
		return &result, nil
	default:
		return nil, nil
	}
}
