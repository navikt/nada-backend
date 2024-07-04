import os
import math
import argparse
import logging
import backoff
import requests


@backoff.on_exception(backoff.expo, (requests.exceptions.RequestException, requests.exceptions.HTTPError), max_tries=8)
def put_with_retries(url, multipart_form_data, token):
    return requests.put(url, headers={"Authorization": f"Bearer {token}"},
                        files=multipart_form_data)


@backoff.on_exception(backoff.expo, (requests.exceptions.RequestException, requests.exceptions.HTTPError), max_tries=8)
def patch_with_retries(url, multipart_form_data, token):
    return requests.patch(url, headers={"Authorization": f"Bearer {token}"},
                          files=multipart_form_data)

logging.basicConfig(level=logging.INFO)

def get_quarto_files(files: list, dirName: str = None):
    for file in os.listdir(dirName):
        if not dirName:
            if not os.path.isfile(file):
                get_quarto_files(files, file)
            else:
                files.append(file)
        else:
            if not os.path.isfile(dirName + "/" + file):
                get_quarto_files(files, dirName + "/" + file)
            else:
                files.append(dirName + "/" + file)


def batch_upload_quarto(
        quarto_id: str = "ddd44fd5-25ac-4154-9f30-049c7c1bae82",
        folder: str = "test_story",
        team_token: str = "cff69c74-7393-4a0e-9102-2f2d01db8705",
        host: str = "localhost:8080",
        path: str = "quarto/update",
        batch_size: int = 10
):
    if not os.getcwd().endswith(folder):
        os.chdir(folder)

    files = []
    get_quarto_files(files)
    logging.info(f"Uploading {len(files)} files in batches of {batch_size}")

    for batch_count in range(math.ceil(len(files) / batch_size)):
        multipart_form_data = {}
        start_batch = batch_count*batch_size
        end_batch = start_batch + batch_size
        for file_path in files[start_batch:end_batch]:
            file_name = os.path.basename(file_path)
            with open(file_path, "rb") as file:
                file_contents = file.read()
                multipart_form_data[file_path] = (file_name, file_contents)

        if batch_count == 0:
            res = put_with_retries(f"http://{host}/{path}/{quarto_id}", multipart_form_data, team_token)
        else:
            res = patch_with_retries(f"http://{host}/{path}/{quarto_id}", multipart_form_data, team_token)


        print(res.request.url)
        print(res.request.body)
        print(res.request.headers)

        res.raise_for_status()

        uploaded = end_batch if end_batch < len(files) else len(files)
        logging.info(f"Uploaded {uploaded}/{len(files)} files")


batch_upload_quarto()
