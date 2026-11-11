package main

import (
	"context"
	"encoding/json"
	"log/slog"
)

// demoSellerID owns the demo ads (no real user behind it).
const demoSellerID = "00000000-0000-4000-8000-000000000001"

type demo struct {
	title, desc, cat, region, city, subcity, cond string
	price                                         float64
	negotiable                                    bool
	attrs                                         string
}

var demoListings = []demo{
	{"Toyota Vitz 2012, automatic, very clean", "Well maintained, new tyres, full service history. ዋጋ በድርድር።", "vehicles/cars", "AA", "Addis Ababa", "Bole", "used", 1850000, true, `{"make":"Toyota","model":"Vitz","year":2012,"mileage_km":98000,"fuel":"petrol","transmission":"automatic"}`},
	{"2 bedroom apartment for rent near CMC", "ባለ 2 መኝታ ቤት፣ 3ኛ ፎቅ፣ ውሃና መብራት 24 ሰዓት።", "property/apartments-rent", "AA", "Addis Ababa", "Yeka", "not_applicable", 35000, true, `{"bedrooms":2,"bathrooms":1,"size_m2":85,"furnished":false}`},
	{"Samsung Galaxy A54 128GB - like new", "Bought 6 months ago, with box and charger.", "electronics/phones", "AA", "Addis Ababa", "Arada", "used", 38000, false, `{"brand":"Samsung","storage_gb":128}`},
	{"Bajaj 3-wheeler (bajaj) for sale", "2019 model, papers complete, ready for work.", "vehicles/motorcycles", "OR", "Adama", "", "used", 420000, true, `{"make":"Bajaj","year":2019}`},
	{"Jebena and coffee ceremony set (handmade)", "Traditional clay jebena, 6 sini cups and tray. ባህላዊ የቡና ማፍያ።", "home/kitchen", "AM", "Bahir Dar", "", "new", 2500, false, `{}`},
	{"HP EliteBook 840 G5, Core i5, 16GB RAM", "SSD 256GB, battery good, ideal for office and students.", "electronics/laptops", "AA", "Addis Ababa", "Kirkos", "used", 42000, true, `{"brand":"HP","ram_gb":16,"storage_gb":256}`},
	{"Habesha kemis, handwoven cotton, size M", "ጥበብ ቀሚስ፣ በእጅ የተሸመነ፣ ለበዓል።", "fashion/women", "AA", "Addis Ababa", "Gullele", "new", 6500, false, `{"size":"M"}`},
	{"Land 200 m² for sale with title deed", "Residential plot, road access, near asphalt.", "property/land", "OR", "Bishoftu", "", "not_applicable", 3200000, true, `{"size_m2":200,"title_deed":true}`},
	{"Accountant wanted (2 years experience)", "Private trading company, Merkato area. Send CV.", "jobs/finance", "AA", "Addis Ababa", "Addis Ketema", "not_applicable", 18000, false, `{"job_type":"full_time","experience_years":2}`},
	{"Solar panel kit 300W with battery", "For homes outside the grid, installation possible.", "electronics/solar", "SI", "Hawassa", "", "new", 27000, true, `{"watts":300}`},
	{"Mountain bike 26\" - good condition", "Shimano gears, new brakes.", "sports/bicycles", "DD", "Dire Dawa", "", "used", 9500, true, `{}`},
	{"Teff 1 quintal, white (magna)", "Direct from farm in Gojjam. ነጭ ማኛ ጤፍ።", "agriculture/grains", "AM", "Debre Markos", "", "new", 14500, false, `{"unit":"quintal","quantity":1}`},
}

// SeedDemo inserts demo ads into an empty database so the web shop is not empty on day one.
func SeedDemo(ctx context.Context, s Store, log *slog.Logger) error {
	n, err := s.Count(ctx)
	if err != nil || n > 0 {
		return err
	}
	for _, d := range demoListings {
		d := d
		attrs := json.RawMessage(d.attrs)
		in := &ListingInput{Title: &d.title, Description: &d.desc, PriceETB: &d.price, Negotiable: &d.negotiable,
			Category: &d.cat, Region: &d.region, City: &d.city, Subcity: &d.subcity, Condition: &d.cond, Attributes: &attrs}
		if _, err := s.Create(ctx, demoSellerID, "", in); err != nil {
			return err
		}
	}
	log.Info("demo listings seeded", "count", len(demoListings))
	return nil
}
