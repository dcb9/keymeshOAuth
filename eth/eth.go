package eth

// https://ethereum.stackexchange.com/questions/17051/how-to-select-a-network-id-or-is-there-a-list-of-network-ids?noredirect=1&lq=1
var networkIDs = []int{
	0,
	1,
	1,
	1,
	2,
	3,
	4,
	8,
	42,
	77,
	99,
	7762959,
}

func IsPrivateNetwork(networkID int) bool {
	for _, id := range networkIDs {
		if id == networkID {
			return false
		}
	}
	return true
}
