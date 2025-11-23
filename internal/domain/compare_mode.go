package domain

type CompareMode uint32

const (
	CompareStrict      CompareMode = 0
	CompareIgnoreIndex CompareMode = 1 << iota
	CompareIgnoreLayout
	CompareIgnoreCommand
	CompareIgnorePath
	CompareIgnoreName
)
