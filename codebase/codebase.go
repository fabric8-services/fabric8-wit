package codebase

import "fmt"

// Codebase defines all parameters those are useful to associate Che Editor's window to a WI
type Codebase struct {
	Repository string `json:"repo"`
	Branch     string `json:"branch"`
	FileName   string `json:"filename"`
	LineNumber int    `json:"linenumber"`
}

const (
	RepositoryKey = "repository"
	BranchKey     = "branch"
	FileNameKey   = "file"
	LineNumberKey = "line"
)

func (c *Codebase) ToMap() map[string]interface{} {
	res := make(map[string]interface{})
	res[RepositoryKey] = c.Repository
	res[BranchKey] = c.Branch
	res[FileNameKey] = c.FileName
	res[LineNumberKey] = c.LineNumber
	return res
}

func NewCodebase(value map[string]interface{}) (Codebase, error) {
	cb := Codebase{}
	validKeys := []string{RepositoryKey, BranchKey, FileNameKey, LineNumberKey}
	for _, key := range validKeys {
		fmt.Println("checking key = ", key)
		if v, ok := value[key]; ok {
			switch key {
			case RepositoryKey:
				cb.Repository = v.(string)
			case BranchKey:
				cb.Branch = v.(string)
			case FileNameKey:
				cb.FileName = v.(string)
			case LineNumberKey:
				cb.LineNumber = v.(int)
			}
		}
	}
	return cb, nil
}
