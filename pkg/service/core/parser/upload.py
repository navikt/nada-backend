import os

import requests

ENV = "localhost:8080"
TEAM_TOKEN = "cff69c74-7393-4a0e-9102-2f2d01db8705"

# res = requests.post(f"http://{ENV}/api/story/create", headers={"Authorization": f"Bearer {TEAM_TOKEN}"}, json={
#     "name": "Min datafortelling om noe",
#     "title": "Min datafortelling om noe",
#     "slug": "min-datafortelling",
#     "group": "nada@nav.no",
# })
# print(res)
# res.raise_for_status()
# story_id = res.json()["id"]
story_id = "ddd44fd5-25ac-4154-9f30-049c7c1bae82"

files_to_upload = [
    "test_story/index.html",
    "test_story/index.css",
    "test_story/subpage/test.html",
    "test_story/subpage/subsubpage/something.html",
]

multipart_form_data = {}
for file_path in files_to_upload:
    file_name = os.path.basename(file_path)
    with open(file_path, 'rb') as file:
        # Read the file contents and store them in the dictionary
        file_contents = file.read()
        multipart_form_data[file_path] = (file_name, file_contents)

# Send the request with all files in the dictionary
response = requests.put( f"http://{ENV}/story/update/{story_id}",
                         headers={"Authorization": f"Bearer {TEAM_TOKEN}"},
                         files=multipart_form_data)

print(response.request.url)
print(response.request.body)
print(response.request.headers)

response.raise_for_status()
