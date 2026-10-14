// A source whose package does not mirror its folder: it sits in fittings/ and declares the tap's
// own package, which the structure lint reads and the order lint, reading its path, does not.
package fixture.features.tap

final case class Washer(worn: Boolean)
