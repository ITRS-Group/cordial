import holidays
import datetime
import sys

thisyear = datetime.datetime.now().year

if len(sys.argv) > 1:
    countries = sys.argv[1:]
else:
    countries = holidays.list_supported_countries(include_aliases=False)
print(countries)

print("country,subdivision,date,holiday")
for country_name in countries:
    country = holidays.country_holidays(country=country_name, years=[thisyear-1, thisyear, thisyear+1], observed=True)
    for date in country:
        print(','.join(map(str, [country_name, "", date, country[date]])))
    # for sub in country.subdivisions:
    #     subdiv = holidays.country_holidays(country=country_name, subdiv=sub, years=[thisyear-1, thisyear, thisyear+1], observed=True)
    #     for day in subdiv:
    #         print(','.join(map(str, [country_name, sub, day, subdiv[day]])))


financials = holidays.list_supported_financial(include_aliases=False)
# print(financials)
