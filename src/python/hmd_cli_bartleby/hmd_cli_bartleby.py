import os
from pathlib import Path
from cement.utils.shell import exec_cmd2
from typing import List, Dict
from hmd_cli_tools.hmd_cli_tools import get_version, cd


hmd_repo_home = os.environ.get("HMD_REPO_HOME")


def transform(
    name: str or List[str],
    version: str,
    transform_instance_context: Dict,
    image_name: str,
    autodoc: bool = False,
):
    repo_path = Path(os.getcwd())

    input_path = repo_path / "docs"
    output_path = repo_path / "target" / "bartleby"

    if not input_path.exists():
        raise Exception("No docs folder found in the current working directory.")

    try:
        if Path(repo_path / "src" / "python").exists() and autodoc:
            pip_username = os.environ.get("PIP_USERNAME")
            pip_password = os.environ.get("PIP_PASSWORD")

            command = [
                "docker",
                "run",
                "--rm",
                "-v",
                f"{input_path}:/hmd_transform/input",
                "-v",
                f"{output_path}:/hmd_transform/output",
                "-e",
                f"TRANSFORM_INSTANCE_CONTEXT={transform_instance_context}",
                "-e",
                f"HMD_DOC_REPO_NAME={name}",
                "-e",
                f"HMD_DOC_REPO_VERSION={version}",
                "-e",
                "AUTODOC=True",
                "-e",
                f"PIP_USERNAME={pip_username}",
                "-e",
                f"PIP_PASSWORD={pip_password}",
                image_name,
            ]

            return_code = exec_cmd2(command)
        else:
            command = [
                "docker",
                "run",
                "--rm",
                "-v",
                f"{input_path}:/hmd_transform/input",
                "-v",
                f"{output_path}:/hmd_transform/output",
                "-e",
                f"TRANSFORM_INSTANCE_CONTEXT={transform_instance_context}",
                "-e",
                f"HMD_DOC_REPO_NAME={name}",
                "-e",
                f"HMD_DOC_REPO_VERSION={version}",
                image_name,
            ]

            return_code = exec_cmd2(command)

        if return_code != 0:
            raise Exception(f"Process completed with non-zero exit code: {return_code}")

    except Exception as e:
        print(f"Exception occurred running: {e}")
