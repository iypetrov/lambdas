package message

type AlarmMessage struct {
	AlarmName                          string       `json:"AlarmName"`
	AlarmDescription                   string       `json:"AlarmDescription"`
	AWSAccountID                       string       `json:"AWSAccountId"`
	AlarmConfigurationUpdatedTimestamp string       `json:"AlarmConfigurationUpdatedTimestamp"`
	NewStateValue                      string       `json:"NewStateValue"`
	NewStateReason                     string       `json:"NewStateReason"`
	StateChangeTime                    string       `json:"StateChangeTime"`
	Region                             string       `json:"Region"`
	AlarmArn                           string       `json:"AlarmArn"`
	OldStateValue                      string       `json:"OldStateValue"`
	OKActions                          []string     `json:"OKActions"`
	AlarmActions                       []string     `json:"AlarmActions"`
	InsufficientDataActions            []string     `json:"InsufficientDataActions"`
	Trigger                            AlarmTrigger `json:"Trigger"`
}

type AlarmTrigger struct {
	MetricName                       string      `json:"MetricName"`
	Namespace                        string      `json:"Namespace"`
	StatisticType                    string      `json:"StatisticType"`
	Statistic                        string      `json:"Statistic"`
	Unit                             *string     `json:"Unit"`
	Dimensions                       []Dimension `json:"Dimensions"`
	Period                           int         `json:"Period"`
	EvaluationPeriods                int         `json:"EvaluationPeriods"`
	ComparisonOperator               string      `json:"ComparisonOperator"`
	Threshold                        float64     `json:"Threshold"`
	TreatMissingData                 string      `json:"TreatMissingData"`
	EvaluateLowSampleCountPercentile string      `json:"EvaluateLowSampleCountPercentile"`
}

type Dimension struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
