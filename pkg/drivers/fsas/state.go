package fsas

// State represents the state of the FSAS host
type CdiMachineState int

const (
	None                  CdiMachineState = iota
	BUILDING_BEFORE_QUEUE CdiMachineState = 10
	BUILDING              CdiMachineState = 11
	BOOTING               CdiMachineState = 12
	ACTIVE_PON            CdiMachineState = 13
	POWERING_OFF          CdiMachineState = 14
	ACTIVE_POFF           CdiMachineState = 15
	UNBUILDING            CdiMachineState = 16
	UNBUILDED             CdiMachineState = 17
	OS_INSTALLING         CdiMachineState = 18
	ERASING               CdiMachineState = 19
	ADDING_RESOURCE       CdiMachineState = 20
	DELETING_RESOURCE     CdiMachineState = 21
	UNBUILDING_WAIT       CdiMachineState = 30
	ERROR                 CdiMachineState = 90
)

// Given a State type, returns its string representation
func (s CdiMachineState) String() string {
	switch s {
	case BUILDING_BEFORE_QUEUE:
		return "BUILDING_BEFORE_QUEUE"
	case BUILDING:
		return "BUILDING"
	case BOOTING:
		return "BOOTING"
	case ACTIVE_PON:
		return "ACTIVE_PON"
	case POWERING_OFF:
		return "POWERING_OFF"
	case ACTIVE_POFF:
		return "ACTIVE_POFF"
	case UNBUILDING:
		return "UNBUILDING"
	case UNBUILDED:
		return "UNBUILDED"
	case OS_INSTALLING:
		return "OS_INSTALLING"
	case ERASING:
		return "ERASING"
	case ADDING_RESOURCE:
		return "ADDING_RESOURCE"
	case DELETING_RESOURCE:
		return "DELETING_RESOURCE"
	case UNBUILDING_WAIT:
		return "UNBUILDING_WAIT"
	case ERROR:
		return "ERROR"
	default:
		return ""
	}
}
