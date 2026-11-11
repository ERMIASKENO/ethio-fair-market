"""Ethiopian administrative locations: regions / chartered cities, major cities, Addis Ababa sub-cities.

Codes are short, stable identifiers stored with every listing (e.g. region "AA", city "Addis Ababa").
"""

from __future__ import annotations

import re
from dataclasses import dataclass, field


def slugify(text: str) -> str:
    return re.sub(r"[^a-z0-9]+", "-", text.lower()).strip("-")


@dataclass(frozen=True)
class City:
    name: str
    name_am: str
    subcities: tuple[tuple[str, str], ...] = ()  # (english, amharic)

    @property
    def slug(self) -> str:
        return slugify(self.name)


@dataclass(frozen=True)
class Region:
    code: str
    name: str
    name_am: str
    kind: str  # "region" or "chartered_city"
    cities: tuple[City, ...] = field(default_factory=tuple)


ADDIS_SUBCITIES = (
    ("Addis Ketema", "አዲስ ከተማ"),
    ("Akaky Kaliti", "አቃቂ ቃሊቲ"),
    ("Arada", "አራዳ"),
    ("Bole", "ቦሌ"),
    ("Gullele", "ጉለሌ"),
    ("Kirkos", "ቂርቆስ"),
    ("Kolfe Keranio", "ኮልፌ ቀራኒዮ"),
    ("Lemi Kura", "ለሚ ኩራ"),
    ("Lideta", "ልደታ"),
    ("Nifas Silk-Lafto", "ንፋስ ስልክ ላፍቶ"),
    ("Yeka", "የካ"),
)

REGIONS: tuple[Region, ...] = (
    Region("AA", "Addis Ababa", "አዲስ አበባ", "chartered_city",
           (City("Addis Ababa", "አዲስ አበባ", ADDIS_SUBCITIES),)),
    Region("DD", "Dire Dawa", "ድሬዳዋ", "chartered_city", (City("Dire Dawa", "ድሬዳዋ"),)),
    Region("AF", "Afar", "አፋር", "region", (City("Semera", "ሰመራ"), City("Asaita", "አሳይታ"), City("Awash", "አዋሽ"))),
    Region("AM", "Amhara", "አማራ", "region", (
        City("Bahir Dar", "ባሕር ዳር"), City("Gondar", "ጎንደር"), City("Dessie", "ደሴ"), City("Debre Markos", "ደብረ ማርቆስ"),
        City("Debre Birhan", "ደብረ ብርሃን"), City("Kombolcha", "ኮምቦልቻ"), City("Woldia", "ወልዲያ"), City("Lalibela", "ላሊበላ"))),
    Region("BG", "Benishangul-Gumuz", "ቤኒሻንጉል ጉሙዝ", "region", (City("Asosa", "አሶሳ"),)),
    Region("CE", "Central Ethiopia", "ማዕከላዊ ኢትዮጵያ", "region", (
        City("Hosaena", "ሆሳዕና"), City("Butajira", "ቡታጅራ"), City("Worabe", "ወራቤ"), City("Welkite", "ወልቂጤ"))),
    Region("GA", "Gambela", "ጋምቤላ", "region", (City("Gambela", "ጋምቤላ"),)),
    Region("HA", "Harari", "ሐረሪ", "region", (City("Harar", "ሐረር"),)),
    Region("OR", "Oromia", "ኦሮሚያ", "region", (
        City("Adama", "አዳማ"), City("Bishoftu", "ቢሾፍቱ"), City("Jimma", "ጅማ"), City("Shashemene", "ሻሸመኔ"),
        City("Nekemte", "ነቀምት"), City("Ambo", "አምቦ"), City("Asella", "አሰላ"), City("Burayu", "ቡራዩ"),
        City("Sebeta", "ሰበታ"), City("Batu", "ባቱ"), City("Robe", "ሮቤ"))),
    Region("SI", "Sidama", "ሲዳማ", "region", (City("Hawassa", "ሀዋሳ"), City("Yirgalem", "ይርጋለም"))),
    Region("SO", "Somali", "ሶማሌ", "region", (City("Jigjiga", "ጅግጅጋ"), City("Gode", "ጎዴ"), City("Degehabur", "ደገሀቡር"))),
    Region("SE", "South Ethiopia", "ደቡብ ኢትዮጵያ", "region", (
        City("Wolaita Sodo", "ወላይታ ሶዶ"), City("Arba Minch", "አርባ ምንጭ"), City("Dilla", "ዲላ"), City("Jinka", "ጂንካ"))),
    Region("SW", "South West Ethiopia Peoples", "ደቡብ ምዕራብ ኢትዮጵያ ሕዝቦች", "region", (
        City("Bonga", "ቦንጋ"), City("Mizan Aman", "ሚዛን አማን"), City("Tepi", "ቴፒ"))),
    Region("TI", "Tigray", "ትግራይ", "region", (
        City("Mekelle", "መቀሌ"), City("Adigrat", "አዲግራት"), City("Axum", "አክሱም"), City("Shire", "ሽሬ"))),
)

BY_CODE: dict[str, Region] = {r.code: r for r in REGIONS}


def find_city(region: Region, key: str) -> City | None:
    """Look up a city by slug ("bahir-dar") or name ("Bahir Dar"), case-insensitive."""
    k = key.strip().lower()
    for c in region.cities:
        if k in (c.slug, c.name.lower()):
            return c
    return None
