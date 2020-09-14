package testutils

import (
	"StaticRaceDetector/domain"
	"StaticRaceDetector/utils"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
)

type TestResult struct {
	Lockset       *domain.Lockset
	GuardedAccess []*domain.GuardedAccess
}

type TestResultJSON struct {
	Lockset       *domain.LocksetJson
	GuardedAccess []*domain.GuardedAccessJSON
}

func WriteResult(t *testing.T, path string, ls *domain.Lockset, ga []*domain.GuardedAccess) {
	testresult := TestResult{Lockset: ls, GuardedAccess: ga}
	dump, err := json.Marshal(testresult)
	require.NoError(t, err)
	utils.UpdateFile(t, path, dump)
}

// CompareResult - CompareResult checks the correctness of the test result against the expected result. It checks if all
// the fields match except for goroutine id. There, it validates all guarded accesses correspond to the same goroutine
// they were associated to before.
// For comparing the guarded accesses 2 types of data structures are created. The first is a map containing all the
// goroutine IDs of the goroutines that were already added and the first instruction in the goroutine. The second is a
// hash table that maps between the first instr (it's pos) in the goroutine to the rest of the instructions. Since an
// instruction can appear in different goroutines and the goroutine id alone doesn't guarantee uniqueness, this form of
// double key was chosen. Then each expected and the actual instructions of each goroutine are compared and make sure
// everything still exists and match.
func CompareResult(t *testing.T, path string, ls *domain.Lockset, ga []*domain.GuardedAccess) {
	testresult := &TestResultJSON{}
	data, err := utils.ReadFile(path)
	require.NoError(t, err)
	err = json.Unmarshal(data, testresult)
	require.NoError(t, err)

	require.Equal(t, ls.ToJson(), testresult.Lockset)

	expectedGuardedAccess := map[int][]*domain.GuardedAccessJSON{}
	expectedGuardedAccessGoroutineIDsSet := map[string]int{}
	for _, guardedAccess := range testresult.GuardedAccess {
		if expectedGuardedAccessKey, ok := expectedGuardedAccessGoroutineIDsSet[guardedAccess.GoroutineId]; !ok {
			guardedAccessPos := guardedAccess.Value
			expectedGuardedAccessGoroutineIDsSet[guardedAccess.GoroutineId] = guardedAccessPos
			expectedGuardedAccess[guardedAccessPos] = append(expectedGuardedAccess[guardedAccessPos], guardedAccess)
		} else {
			expectedGuardedAccess[expectedGuardedAccessKey] = append(expectedGuardedAccess[expectedGuardedAccessKey], guardedAccess)
		}
	}

	actualGuardedAccess := map[int][]*domain.GuardedAccess{}
	actualGuardedAccessGoroutineIDsSet := map[string]int{}
	for _, guardedAccess := range ga {
		if expectedGuardedAccessKey, ok := actualGuardedAccessGoroutineIDsSet[guardedAccess.GoroutineId]; !ok {
			guardedAccessPos := int(guardedAccess.Value.Pos())
			actualGuardedAccessGoroutineIDsSet[guardedAccess.GoroutineId] = guardedAccessPos
			actualGuardedAccess[guardedAccessPos] = append(actualGuardedAccess[guardedAccessPos], guardedAccess)
		} else {
			actualGuardedAccess[expectedGuardedAccessKey] = append(actualGuardedAccess[expectedGuardedAccessKey], guardedAccess)
		}
	}

	require.Equal(t, len(actualGuardedAccess), len(expectedGuardedAccess)) // Same amount of goroutines were generated
	for key := range expectedGuardedAccess {
		expectedGoroutineInstructions := expectedGuardedAccess[key]
		actualGoroutineInstructions := actualGuardedAccess[key]
		require.Equal(t, len(actualGoroutineInstructions), len(expectedGoroutineInstructions)) // Same amount of instructions in each goroutine
		for i := 0; i < len(expectedGuardedAccess[key]); i++ {
			insr := actualGoroutineInstructions[i].ToJson()
			require.Equal(t, insr.Value, expectedGoroutineInstructions[i].Value)
			require.Equal(t, insr.Lockset, expectedGoroutineInstructions[i].Lockset)
			require.Equal(t, insr.OpKind, expectedGoroutineInstructions[i].OpKind)
		}
	}
}
