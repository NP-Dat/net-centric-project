package game

// CalculateDamage calculates the damage dealt based on attacker's ATK and defender's DEF.
// Damage Formula: DMG = Attacker_ATK - Defender_DEF. If DMG < 0, DMG = 0.
func CalculateDamage(attackerATK, defenderDEF int) int {
	damage := attackerATK - defenderDEF
	if damage < 0 {
		return 0
	}
	return damage
}
