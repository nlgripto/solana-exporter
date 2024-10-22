package main

import (
	"context"
	"fmt"
	"github.com/asymmetric-research/solana_exporter/pkg/rpc"
	"k8s.io/klog/v2"
	"slices"
)

func assertf(condition bool, format string, args ...any) {
	if !condition {
		klog.Fatalf(format, args...)
	}
}

// toString is just a simple utility function for converting int -> string
func toString(i int64) string {
	return fmt.Sprintf("%v", i)
}

// SelectFromSchedule takes a leader-schedule and returns a trimmed leader-schedule
// containing only the slots within the provided range
func SelectFromSchedule(schedule map[string][]int64, startSlot, endSlot int64) map[string][]int64 {
	selected := make(map[string][]int64)
	for key, values := range schedule {
		var selectedValues []int64
		for _, value := range values {
			if value >= startSlot && value <= endSlot {
				selectedValues = append(selectedValues, value)
			}
		}
		selected[key] = selectedValues
	}
	return selected
}

// GetTrimmedLeaderSchedule fetches the leader schedule, but only for the validators we are interested in.
// Additionally, it adjusts the leader schedule to the current epoch offset.
func GetTrimmedLeaderSchedule(
	ctx context.Context, client rpc.Provider, identities []string, slot, epochFirstSlot int64,
) (map[string][]int64, error) {
	leaderSchedule, err := client.GetLeaderSchedule(ctx, rpc.CommitmentConfirmed, slot)
	if err != nil {
		return nil, fmt.Errorf("failed to get leader schedule: %w", err)
	}

	trimmedLeaderSchedule := make(map[string][]int64)
	for _, id := range identities {
		if leaderSlots, ok := leaderSchedule[id]; ok {
			// when you fetch the leader schedule, it gives you slot indexes, we want absolute slots:
			absoluteSlots := make([]int64, len(leaderSlots))
			for i, slotIndex := range leaderSlots {
				absoluteSlots[i] = slotIndex + epochFirstSlot
			}
			trimmedLeaderSchedule[id] = absoluteSlots
		} else {
			klog.Warningf("failed to find leader slots for %v", id)
		}
	}

	return trimmedLeaderSchedule, nil
}

// GetAssociatedVoteAccounts returns the votekeys associated with a given list of nodekeys
func GetAssociatedVoteAccounts(
	ctx context.Context, client rpc.Provider, commitment rpc.Commitment, nodekeys []string,
) ([]string, error) {
	voteAccounts, err := client.GetVoteAccounts(ctx, commitment, nil)
	if err != nil {
		return nil, err
	}

	// first map nodekey -> votekey:
	voteAccountsMap := make(map[string]string)
	for _, voteAccount := range append(voteAccounts.Current, voteAccounts.Delinquent...) {
		voteAccountsMap[voteAccount.NodePubkey] = voteAccount.VotePubkey
	}

	votekeys := make([]string, len(nodekeys))
	for i, nodeKey := range nodekeys {
		votekey := voteAccountsMap[nodeKey]
		if votekey == "" {
			return nil, fmt.Errorf("failed to find vote key for node %v", nodeKey)
		}
		votekeys[i] = votekey
	}
	return votekeys, nil
}

// FetchBalances fetches SOL balances for a list of addresses
func FetchBalances(ctx context.Context, client rpc.Provider, addresses []string) (map[string]float64, error) {
	balances := make(map[string]float64)
	for _, address := range addresses {
		balance, err := client.GetBalance(ctx, rpc.CommitmentConfirmed, address)
		if err != nil {
			return nil, err
		}
		balances[address] = balance
	}
	return balances, nil
}

// CombineUnique combines unique items from multiple arrays to a single array.
func CombineUnique[T comparable](args ...[]T) []T {
	var uniqueItems []T
	for _, arg := range args {
		for _, item := range arg {
			if !slices.Contains(uniqueItems, item) {
				uniqueItems = append(uniqueItems, item)
			}
		}
	}
	return uniqueItems
}