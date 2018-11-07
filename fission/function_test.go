package main

import (
	"flag"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"

	"github.com/fission/fission"
)

func TestGetInvokeStrategy(t *testing.T) {
	cases := []struct {
		testArgs               map[string]string
		existingInvokeStrategy *fission.InvokeStrategy
		expectedResult         *fission.InvokeStrategy
		expectError            bool
	}{
		{
			// case: use default executor poolmgr
			testArgs:               map[string]string{},
			existingInvokeStrategy: nil,
			expectedResult: &fission.InvokeStrategy{
				StrategyType: fission.StrategyTypeExecution,
				ExecutionStrategy: fission.ExecutionStrategy{
					ExecutorType: fission.ExecutorTypePoolmgr,
				},
			},
			expectError: false,
		},
		{
			// case: executor type set to poolmgr
			testArgs:               map[string]string{"executortype": fission.ExecutorTypePoolmgr},
			existingInvokeStrategy: nil,
			expectedResult: &fission.InvokeStrategy{
				StrategyType: fission.StrategyTypeExecution,
				ExecutionStrategy: fission.ExecutionStrategy{
					ExecutorType: fission.ExecutorTypePoolmgr,
				},
			},
			expectError: false,
		},
		{
			// case: executor type set to newdeploy
			testArgs:               map[string]string{"executortype": fission.ExecutorTypeNewdeploy},
			existingInvokeStrategy: nil,
			expectedResult: &fission.InvokeStrategy{
				StrategyType: fission.StrategyTypeExecution,
				ExecutionStrategy: fission.ExecutionStrategy{
					ExecutorType:     fission.ExecutorTypeNewdeploy,
					MinScale:         DEFAULT_MIN_SCALE,
					MaxScale:         DEFAULT_MIN_SCALE,
					TargetCPUPercent: DEFAULT_TARGET_CPU_PERCENTAGE,
				},
			},
			expectError: false,
		},
		{
			// case: executor type change from poolmgr to newdeploy
			testArgs: map[string]string{"executortype": fission.ExecutorTypeNewdeploy},
			existingInvokeStrategy: &fission.InvokeStrategy{
				StrategyType: fission.StrategyTypeExecution,
				ExecutionStrategy: fission.ExecutionStrategy{
					ExecutorType: fission.ExecutorTypePoolmgr,
				},
			},
			expectedResult: &fission.InvokeStrategy{
				StrategyType: fission.StrategyTypeExecution,
				ExecutionStrategy: fission.ExecutionStrategy{
					ExecutorType:     fission.ExecutorTypeNewdeploy,
					MinScale:         DEFAULT_MIN_SCALE,
					MaxScale:         DEFAULT_MIN_SCALE,
					TargetCPUPercent: DEFAULT_TARGET_CPU_PERCENTAGE,
				},
			},
			expectError: false,
		},
		{
			// case: executor type change from newdeploy to poolmgr
			testArgs: map[string]string{"executortype": fission.ExecutorTypePoolmgr},
			existingInvokeStrategy: &fission.InvokeStrategy{
				StrategyType: fission.StrategyTypeExecution,
				ExecutionStrategy: fission.ExecutionStrategy{
					ExecutorType:     fission.ExecutorTypeNewdeploy,
					MinScale:         DEFAULT_MIN_SCALE,
					MaxScale:         DEFAULT_MIN_SCALE,
					TargetCPUPercent: DEFAULT_TARGET_CPU_PERCENTAGE,
				},
			},
			expectedResult: &fission.InvokeStrategy{
				StrategyType: fission.StrategyTypeExecution,
				ExecutionStrategy: fission.ExecutionStrategy{
					ExecutorType: fission.ExecutorTypePoolmgr,
				},
			},
			expectError: false,
		},
		{
			// case: minscale < maxscale
			testArgs: map[string]string{
				"executortype": fission.ExecutorTypeNewdeploy,
				"minscale":     "2",
				"maxscale":     "3",
			},
			existingInvokeStrategy: nil,
			expectedResult: &fission.InvokeStrategy{
				StrategyType: fission.StrategyTypeExecution,
				ExecutionStrategy: fission.ExecutionStrategy{
					ExecutorType:     fission.ExecutorTypeNewdeploy,
					MinScale:         2,
					MaxScale:         3,
					TargetCPUPercent: DEFAULT_TARGET_CPU_PERCENTAGE,
				},
			},
			expectError: false,
		},
		{
			// case: minscale > maxscale
			testArgs: map[string]string{
				"executortype": fission.ExecutorTypeNewdeploy,
				"minscale":     "5",
				"maxscale":     "3",
			},
			existingInvokeStrategy: nil,
			expectedResult:         nil,
			expectError:            true,
		},
		{
			// case: maxscale not specified
			testArgs: map[string]string{
				"executortype": fission.ExecutorTypeNewdeploy,
				"minscale":     "5",
			},
			existingInvokeStrategy: nil,
			expectedResult:         nil,
			expectError:            true,
		},
		{
			// case: minscale not specified
			testArgs: map[string]string{
				"executortype": fission.ExecutorTypeNewdeploy,
				"maxscale":     "3",
			},
			existingInvokeStrategy: nil,
			expectedResult: &fission.InvokeStrategy{
				StrategyType: fission.StrategyTypeExecution,
				ExecutionStrategy: fission.ExecutionStrategy{
					ExecutorType:     fission.ExecutorTypeNewdeploy,
					MinScale:         DEFAULT_MIN_SCALE,
					MaxScale:         3,
					TargetCPUPercent: DEFAULT_TARGET_CPU_PERCENTAGE,
				},
			},
			expectError: false,
		},
		{
			// case: maxscale set to 0
			testArgs: map[string]string{
				"executortype": fission.ExecutorTypeNewdeploy,
				"maxscale":     "0",
			},
			existingInvokeStrategy: nil,
			expectedResult:         nil,
			expectError:            true,
		},
		{
			// case: maxscale set to 9 when existing is 5
			testArgs: map[string]string{
				"executortype": fission.ExecutorTypeNewdeploy,
				"maxscale":     "9",
			},
			existingInvokeStrategy: &fission.InvokeStrategy{
				StrategyType: fission.StrategyTypeExecution,
				ExecutionStrategy: fission.ExecutionStrategy{
					ExecutorType:     fission.ExecutorTypeNewdeploy,
					MinScale:         2,
					MaxScale:         5,
					TargetCPUPercent: DEFAULT_TARGET_CPU_PERCENTAGE,
				},
			},
			expectedResult: &fission.InvokeStrategy{
				StrategyType: fission.StrategyTypeExecution,
				ExecutionStrategy: fission.ExecutionStrategy{
					ExecutorType:     fission.ExecutorTypeNewdeploy,
					MinScale:         2,
					MaxScale:         9,
					TargetCPUPercent: DEFAULT_TARGET_CPU_PERCENTAGE,
				},
			},
			expectError: false,
		},
		{
			// case: change nothing for existing strategy
			testArgs: map[string]string{
				"executortype": fission.ExecutorTypeNewdeploy,
			},
			existingInvokeStrategy: &fission.InvokeStrategy{
				StrategyType: fission.StrategyTypeExecution,
				ExecutionStrategy: fission.ExecutionStrategy{
					ExecutorType:     fission.ExecutorTypeNewdeploy,
					MinScale:         2,
					MaxScale:         5,
					TargetCPUPercent: DEFAULT_TARGET_CPU_PERCENTAGE,
				},
			},
			expectedResult: &fission.InvokeStrategy{
				StrategyType: fission.StrategyTypeExecution,
				ExecutionStrategy: fission.ExecutionStrategy{
					ExecutorType:     fission.ExecutorTypeNewdeploy,
					MinScale:         2,
					MaxScale:         5,
					TargetCPUPercent: DEFAULT_TARGET_CPU_PERCENTAGE,
				},
			},
			expectError: false,
		},
		{
			// case: set target cpu percentage
			testArgs: map[string]string{
				"executortype": fission.ExecutorTypeNewdeploy,
				"targetcpu":    "50",
			},
			existingInvokeStrategy: nil,
			expectedResult: &fission.InvokeStrategy{
				StrategyType: fission.StrategyTypeExecution,
				ExecutionStrategy: fission.ExecutionStrategy{
					ExecutorType:     fission.ExecutorTypeNewdeploy,
					MinScale:         DEFAULT_MIN_SCALE,
					MaxScale:         DEFAULT_MIN_SCALE,
					TargetCPUPercent: 50,
				},
			},
			expectError: false,
		},
		{
			// case: change target cpu percentage
			testArgs: map[string]string{
				"executortype": fission.ExecutorTypeNewdeploy,
				"targetcpu":    "20",
			},
			existingInvokeStrategy: &fission.InvokeStrategy{
				StrategyType: fission.StrategyTypeExecution,
				ExecutionStrategy: fission.ExecutionStrategy{
					ExecutorType:     fission.ExecutorTypeNewdeploy,
					MinScale:         2,
					MaxScale:         5,
					TargetCPUPercent: 88,
				},
			},
			expectedResult: &fission.InvokeStrategy{
				StrategyType: fission.StrategyTypeExecution,
				ExecutionStrategy: fission.ExecutionStrategy{
					ExecutorType:     fission.ExecutorTypeNewdeploy,
					MinScale:         2,
					MaxScale:         5,
					TargetCPUPercent: 20,
				},
			},
			expectError: false,
		},
	}

	for i, c := range cases {
		fmt.Printf("=== Test Case %v ===\n", i)

		app := newCliApp()
		set := flag.NewFlagSet("test-cmd", 0)
		ctx := cli.NewContext(app, set, nil)

		for k, v := range c.testArgs {
			set.String(k, v, "")
			ctx.Set(k, v)
		}

		strategy, err := getInvokeStrategy(ctx, c.existingInvokeStrategy)
		if c.expectError {
			assert.NotNil(t, err)
			if err != nil {
				fmt.Println(err)
			}
		} else {
			assert.Nil(t, err)
			assert.NoError(t, strategy.Validate(), fmt.Sprintf("Failed at test case %v", i))
			assert.Equal(t, *c.expectedResult, *strategy)
		}
	}
}
