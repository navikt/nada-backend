# Prereq
# gcloud auth application-default login

from google.cloud import datacatalog_v1

dc = datacatalog_v1.DataCatalogClient()
lookup_req = datacatalog_v1.types.LookupEntryRequest(
            linked_resource=f"//bigquery.googleapis.com/projects/dataplattform-dev-9da3/datasets/ereg/tables/ereg"
)

table_metadata = dc.lookup_entry(lookup_req)

print(table_metadata)
