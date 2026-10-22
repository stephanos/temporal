package kettle

const MaxKettleIDBytes = 32

func maxTags(kettleID string) int {
	if kettleID == "compact" {
		return 2
	}
	return 4
}
