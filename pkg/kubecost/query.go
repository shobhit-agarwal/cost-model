package kubecost

import (
	"time"
)

// Querier is an aggregate interface which has the ability to query each Kubecost store type
type Querier interface {
	AllocationQuerier
	SummaryAllocationQuerier
	AssetQuerier
	CloudUsageQuerier
}

// AllocationQuerier interface defining api for requesting Allocation data
type AllocationQuerier interface {
	QueryAllocation(start, end time.Time, opts *AllocationQueryOptions) (chan *AllocationSetRange, chan error)
	QueryAllocationSync(start, end time.Time, opts *AllocationQueryOptions) (*AllocationSetRange, error)
}

// SummaryAllocationQuerier interface defining api for requesting SummaryAllocation data
type SummaryAllocationQuerier interface {
	QuerySummaryAllocation(start, end time.Time, opts *AllocationQueryOptions) (chan *SummaryAllocationSetRange, chan error)
	QuerySummaryAllocationSync(start, end time.Time, opts *AllocationQueryOptions) (*SummaryAllocationSetRange, error)
}

// AssetQuerier interface defining api for requesting Asset data
type AssetQuerier interface {
	QueryAsset(start, end time.Time, opts *AssetQueryOptions) (*AssetSetRange, error)
}

// CloudUsageQuerier interface defining api for requesting CloudUsage data
type CloudUsageQuerier interface {
	QueryCloudUsage(start, end time.Time, opts *CloudUsageQueryOptions) (*CloudUsageSetRange, error)
}

// AllocationQueryOptions defines optional parameters for querying an Allocation Store
type AllocationQueryOptions struct {
	Accumulate        bool
	AccumulateBy      time.Duration
	AggregateBy       []string
	Compute           bool
	FilterFuncs       []AllocationMatchFunc
	IdleByNode        bool
	IncludeExternal   bool
	IncludeIdle       bool
	LabelConfig       *LabelConfig
	MergeUnallocated  bool
	Reconcile         bool
	ReconcileNetwork  bool
	ShareFuncs        []AllocationMatchFunc
	SharedHourlyCosts map[string]float64
	ShareIdle         string
	ShareSplit        string
	ShareTenancyCosts bool
	SplitIdle         bool
	Step              time.Duration
}

// AssetQueryOptions defines optional parameters for querying an Asset Store
type AssetQueryOptions struct {
	Accumulate         bool
	AggregateBy        []string
	AwaitCoverage      bool
	Compute            bool
	DisableAdjustments bool
	FilterFuncs        []AssetMatchFunc
	ShareFuncs         []AssetMatchFunc
	SharedHourlyCosts  map[string]float64
	Step               time.Duration
}

// CloudUsageQueryOptions define optional parameters for querying a Store
type CloudUsageQueryOptions struct {
	Accumulate    bool
	AggregateBy   []string
	AwaitCoverage bool
	FilterFuncs   []CloudUsageMatchFunc
	Step          time.Duration
}

// QueryAllocationAsync provide a functions for retrieving results from any AllocationQuerier Asynchronously
func QueryAllocationAsync(allocationQuerier *AllocationQuerier, start, end time.Time, opts *AllocationQueryOptions) (chan *AllocationSetRange, chan error) {
	asrCh := make(chan *AllocationSetRange)
	errCh := make(chan error)

	go func(asrCh chan *AllocationSetRange, errCh chan error) {
		defer close(asrCh)
		defer close(errCh)

		asr, err := allocationQuerier.QueryAllocationSync(start, end, opts)
		if err != nil {
			errCh <- err
			return
		}

		asrCh <- asr
	}(asrCh, errCh)

	return asrCh, errCh
}

// QuerySummaryAllocationAsync provide a functions for retrieving results from any SummaryAllocationQuerier Asynchronously
func QuerySummaryAllocationAsync(summaryAllocationQuerier *SummaryAllocationQuerier, start, end time.Time, opts *AllocationQueryOptions) (chan *SummaryAllocationSetRange, chan error) {
	asrCh := make(chan *SummaryAllocationSetRange)
	errCh := make(chan error)

	go func(asrCh chan *SummaryAllocationSetRange, errCh chan error) {
		defer close(asrCh)
		defer close(errCh)

		asr, err := summaryAllocationQuerier.QuerySummaryAllocationSync(start, end, opts)
		if err != nil {
			errCh <- err
			return
		}

		asrCh <- asr
	}(asrCh, errCh)

	return asrCh, errCh
}

// QueryAsseetAsync provide a functions for retrieving results from any AssetQuerier Asynchronously
func QueryAssetAsync(assetQuerier *AssetQuerier, start, end time.Time, opts *AssetQueryOptions) (chan *AssetSetRange, chan error) {
	asrCh := make(chan *AssetSetRange)
	errCh := make(chan error)

	go func(asrCh chan *AssetSetRange, errCh chan error) {
		defer close(asrCh)
		defer close(errCh)

		asr, err := assetQuerier.QueryAsset(start, end, opts)
		if err != nil {
			errCh <-  err
			return
		}

		asrCh <- asr
	} (asrCh, errCh)

	return asrCh, errCh
}

// QueryCloudUsage provide a functions for retrieving results from any CloudUsageQuerier Asynchronously
func QueryCloudUsage(cloudUsageQuerier CloudUsageQuerier, start, end time.Time, opts *CloudUsageQueryOptions) (chan *CloudUsageSetRange, chan error) {
	cusrCh := make(chan *CloudUsageSetRange)
	errCh := make(chan error)
	go func(cusrCh chan *CloudUsageSetRange, errCh chan error) {
		defer close(cusrCh)
		defer close(errCh)
		cusr, err := cloudUsageQuerier.QueryCloudUsage(start, end, opts)
		if err != nil {
			errCh <- err
			return
		}

		cusrCh <- cusr
	}(cusrCh, errCh)
	return cusrCh, errCh
}