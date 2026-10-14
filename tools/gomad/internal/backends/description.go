package backends

import "go.temporal.io/server/tools/gomad/internal/evidence/record"

func (description Description) Name() string { return description.IOProfile.Name }
func (description Description) ImplementationSHA256() string {
	return string(description.IOProfile.ImplementationSHA256)
}
func (description Description) InventorySHA256() string {
	return string(description.IOProfile.InventorySHA256)
}
func (description Description) Inventory() string { return description.IOProfile.Inventory }
func (description Description) Identity() record.IOContract {
	return record.IOContract{Name: description.IOProfile.Name, ImplementationSHA256: description.IOProfile.ImplementationSHA256, InventorySHA256: description.IOProfile.InventorySHA256}
}
func (description Description) Matches(identity record.IOContract) bool {
	return description.Identity() == identity
}
func (description Description) MatchesRecorded(name, implementation, inventorySHA256, inventory string) bool {
	return name == description.Name() && implementation == description.ImplementationSHA256() && inventorySHA256 == description.InventorySHA256() && inventory == description.Inventory()
}
