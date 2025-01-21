package trigger

type SampleList []SampleData

// GetResourcePairs for search.
// pairs[0] is sampleID, pairs[1] is sampleVersion.
func (l SampleList) GetResourcePairs() (pairs [][2]string) {
	for _, sample := range l {
		pairs = append(pairs, [2]string{sample.GetID(), sample.GetVersion()})
	}
	return
}

type SampleData interface {
	GetID() string
	GetVersion() string
}

type SampleProvider interface {
	// GetConfigObject get the trigger config object
	GetConfigObject() any

	// GetSampleList returns a list of samples.
	// index 0 data should be latest.
	GetSampleList(ctx *TriggerDeps) (result []SampleData, err error)
}
