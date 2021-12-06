import pathlib

from setuptools import find_packages, setup

repo_dir = pathlib.Path(__file__).absolute().parent.parent.parent
version_file = repo_dir / "meta-data" / "VERSION"

with open(version_file, "r") as vfl:
    version = vfl.read().strip()

setup(
    name="hmd-cli-bartleby",
    version=version,
    description="CLI for bartleby transform",
    author="Kate Walls",
    author_email="kate.walls@hmdlabs.io",
    license="unlicensed",
    packages=find_packages(),
    include_package_data=True,
    install_requires=[],
)
