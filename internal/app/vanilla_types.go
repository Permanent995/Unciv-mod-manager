package app

// Vanilla types/promotions/terrains — Civ5 engine hardcoded values.
// Unciv loads these from Kotlin enums, not from JSON.
// Civ5 base set is stable — additions are extremely rare.

var vanillaUnitTypes = map[string]bool{
	"Melee": true, "Sword": true, "Spear": true, "Archery": true,
	"Mounted": true, "Armor": true, "RangedGunpowder": true, "Gunpowder": true,
	"WaterMelee": true, "WaterRanged": true, "WaterSubmarine": true, "WaterAircraftCarrier": true,
	"Melee Water": true,
	"Fighter": true, "Bomber": true, "Helicopter": true,
	"Civilian": true, "Civilian Water": true, "Scout": true,
	"Missile": true, "City": true,
}

var vanillaTechs = map[string]bool{
	"Agriculture": true, "Animal Husbandry": true, "Archery": true,
	"Bronze Working": true, "Calendar": true, "Civil Service": true,
	"Compass": true, "Construction": true, "Currency": true,
	"Education": true, "Engineering": true, "Guilds": true,
	"Horseback Riding": true, "Iron Working": true, "Machinery": true,
	"Mathematics": true, "Mining": true, "Masonry": true,
	"Optics": true, "Philosophy": true, "Physics": true,
	"Pottery": true, "Sailing": true, "Steel": true,
	"The Wheel": true, "Theology": true, "Trapping": true,
	"Writing": true, "Acoustics": true, "Astronomy": true,
	"Banking": true, "Chivalry": true, "Drama and Poetry": true,
	"Economics": true, "Fertilizer": true, "Gunpowder": true,
	"Industrialization": true, "Metallurgy": true, "Military Science": true,
	"Military Tradition": true, "Navigation": true, "Printing Press": true,
	"Rifling": true, "Scientific Theory": true, "Steam Power": true,
	"Telegraph": true, "Archaeology": true, "Biology": true,
	"Chemistry": true, "Combustion": true, "Dynamite": true,
	"Ecology": true, "Electricity": true, "Electronics": true,
	"Flight": true, "Mass Media": true, "Nuclear Fission": true,
	"Penicillin": true, "Plastics": true, "Radar": true,
	"Radio": true, "Railroad": true, "Refrigeration": true,
	"Replaceable Parts": true, "Robotics": true, "Rocketry": true,
	"Satellites": true, "Telecommunications": true, "Atomic Theory": true,
	"Ballistics": true, "Combined Arms": true, "Computers": true,
	"Globalization": true, "Lasers": true, "Mobile Tactics": true,
	"Nuclear Fusion": true, "Particle Physics": true, "Stealth": true,
	"Advanced Ballistics": true, "Future Tech": true,
}

var vanillaResources = map[string]bool{
	"Horses": true, "Iron": true, "Coal": true, "Oil": true,
	"Aluminum": true, "Uranium": true,
	"Cattle": true, "Sheep": true, "Deer": true, "Wheat": true,
	"Bananas": true, "Fish": true, "Stone": true,
	"Whales": true, "Pearls": true, "Gold": true, "Silver": true,
	"Gems": true, "Marble": true, "Ivory": true, "Furs": true,
	"Dyes": true, "Incense": true, "Cotton": true, "Silk": true,
	"Spices": true, "Wine": true, "Sugar": true, "Salt": true,
	"Citrus": true, "Truffles": true, "Crab": true, "Copper": true,
	"Bison": true, "Cocoa": true,
}

var vanillaPromotions = map[string]bool{
	"Accuracy I": true, "Accuracy II": true, "Accuracy III": true,
	"Barrage I": true, "Barrage II": true, "Barrage III": true,
	"Shock I": true, "Shock II": true, "Shock III": true,
	"Drill I": true, "Drill II": true, "Drill III": true,
	"Cover I": true, "Cover II": true,
	"Medic I": true, "Medic II": true, "Medic": true,
	"March": true, "Blitz": true, "Logistics": true, "Range": true,
	"Sentry": true, "Mobility": true, "Scouting": true,
	"Siege": true, "Volley": true, "Formation I": true, "Formation II": true,
	"Ambush I": true, "Ambush II": true,
	"Charge": true, "Discipline": true, "Morale": true,
	"Great Generals I": true, "Great Generals II": true,
	"Heal Instantly": true, "Instant Heal": true,
	"Repair": true, "Air Repair": true,
	"Interception I": true, "Interception II": true, "Interception III": true,
	"Dogfighting I": true, "Dogfighting II": true, "Dogfighting III": true,
	"Siege I": true, "Siege II": true, "Siege III": true,
	"Bombardment I": true, "Bombardment II": true, "Bombardment III": true,
	"Targeting I": true, "Targeting II": true, "Targeting III": true,
	"Coastal Raider I": true, "Coastal Raider II": true, "Coastal Raider III": true,
	"Boarding Party I": true, "Boarding Party II": true, "Boarding Party III": true,
	"Supply": true, "Indirect Fire": true,
	"Woodsman": true, "Amphibious": true,
	"Embarkation": true, "Defense": true, "Himeji Castle": true,
	"Recon": true, "Survivalism I": true, "Survivalism II": true, "Survivalism III": true,
	"Naval Tradition": true, "Prize Ships": true,
	"Statue of Zeus": true, "Nationalism": true, "Heroism": true,
	"Wolfpack I": true, "Wolfpack II": true, "Wolfpack III": true,
	"Armor Plating I": true, "Armor Plating II": true, "Armor Plating III": true,
	"Flight Deck I": true, "Flight Deck II": true, "Flight Deck III": true,
	"Evasion": true, "Sortie": true, "Air Ambush": true,
	"Ignore terrain cost": true, "Quick Study": true,
}

var vanillaTerrains = map[string]bool{
	"Grassland": true, "Plains": true, "Desert": true, "Tundra": true, "Snow": true,
	"Coast": true, "Ocean": true, "Lake": true,
	"Hill": true, "Mountain": true,
	"Forest": true, "Jungle": true, "Marsh": true, "Flood plains": true,
	"Oasis": true, "Ice": true, "Atoll": true,
	"Fallout": true, "City Ruins": true,
}

// vanillaEntityNames lists base game entities commonly referenced by
// replaces / upgradesTo in mods.  These are defined in the core Civ V ruleset.
var vanillaEntityNames = map[string]bool{
	// Units
	"Warrior": true, "Scout": true, "Archer": true, "Slinger": true,
	"Spearman": true, "Pikeman": true, "Lancer": true, "Anti-Tank Gun": true,
	"Swordsman": true, "Longswordsman": true, "Musketman": true,
	"Chariot Archer": true, "Horseman": true, "Knight": true, "Cavalry": true, "Landship": true, "Tank": true,
	"Catapult": true, "Trebuchet": true, "Cannon": true, "Artillery": true, "Rocket Artillery": true,
	"Crossbowman": true, "Gatling Gun": true, "Machine Gun": true,
	"Work Boats": true, "Work Boat": true, "Trireme": true, "Galley": true,
	"Caravel": true, "Frigate": true, "Ship of the Line": true, "Privateer": true,
	"Destroyer": true, "Battleship": true, "Ironclad": true, "Submarine": true, "Carrier": true,
	"Great War Bomber": true, "Bomber": true, "Fighter": true, "Triplane": true, "Zero": true,
	"Settler": true, "Worker": true, "Missionary": true, "Great Scientist": true,
	"Great Engineer": true, "Great Merchant": true, "Great Artist": true, "Great Writer": true,
	"Great Musician": true, "Great Prophet": true, "General": true, "Great General": true,
	"Great Admiral": true, "Inquisitor": true, "Archaeologist": true,
	// Buildings
	"Monument": true, "Granary": true, "Library": true, "National College": true,
	"Workshop": true, "Factory": true, "University": true, "Public School": true, "Research Lab": true,
	"Market": true, "Bank": true, "Stock Exchange": true,
	"Barracks": true, "Armory": true, "Military Academy": true,
	"Stable": true, "Forge": true, "Aqueduct": true, "Hospital": true, "Medical Lab": true,
	"Shrine": true, "Temple": true, "Garden": true,
	"Walls": true, "Castle": true, "Arsenal": true, "Military Base": true,
	"Courthouse": true, "Colosseum": true, "Theater": true, "Opera House": true, "Museum": true, "Stadium": true,
	"Caravansary": true, "Harbor": true, "Seaport": true, "Lighthouse": true,
	"Windmill": true, "Hydro Plant": true, "Solar Plant": true, "Nuclear Plant": true,
	"Constabulary": true, "Police Station": true,
	"Airport": true, "Recycling Center": true, "Spaceport": true,
	// Wonders
	"Pyramids": true, "Great Wall": true, "Stonehenge": true, "Oracle": true,
	"Colossus": true, "Great Library": true, "Hanging Gardens": true,
	"Statue of Zeus": true, "Temple of Artemis": true, "Mausoleum of Halicarnassus": true,
	"Terracotta Army": true, "Great Lighthouse": true, "Parthenon": true,
	"Chichen Itza": true, "Angkor Wat": true, "Hagia Sophia": true, "Alhambra": true,
	"Forbidden Palace": true, "Porcelain Tower": true, "Machu Picchu": true,
	"Big Ben": true, "Taj Mahal": true, "Leaning Tower of Pisa": true,
	"Globe Theatre": true, "Sistine Chapel": true, "Uffizi": true, "Louvre": true,
	"Brandenburg Gate": true, "Kremlin": true, "Eiffel Tower": true,
	"Statue of Liberty": true, "Cristo Redentor": true, "Broadway": true,
	"Rock of Cashel": true, "Pentagon": true, "Manhattan Project": true,
	"Apollo Program": true, "Hubble": true, "International Space Station": true,
	"CN Tower": true, "Neuschwanstein": true, "Sydney Opera House": true,
}

// IsVanillaType reports whether a name is a hardcoded game value.
func IsVanillaType(name string) bool {
	return vanillaUnitTypes[name] || vanillaTechs[name] || vanillaResources[name] ||
		vanillaPromotions[name] || vanillaTerrains[name] || vanillaEntityNames[name]
}
