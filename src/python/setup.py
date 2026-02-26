import pathlib
import shutil

from setuptools import find_packages, setup

repo_dir = pathlib.Path(__file__).absolute().parent.parent.parent
version_file = repo_dir / "meta-data" / "VERSION"

with open(version_file, "r") as vfl:
    version = vfl.read().strip()

# Copy agents from src/agents/ into the package for distribution
src_agents = repo_dir / "src" / "agents"
pkg_agents = pathlib.Path(__file__).parent / "hmd_cli_bartleby" / "agents"

if src_agents.exists() and src_agents.is_dir():
    # Remove existing agents dir (or symlink) and copy fresh
    if pkg_agents.is_symlink():
        pkg_agents.unlink()
    elif pkg_agents.exists():
        shutil.rmtree(pkg_agents)
    shutil.copytree(src_agents, pkg_agents)

# Copy skills from src/skills/ into the package for distribution
src_skills = repo_dir / "src" / "skills"
pkg_skills = pathlib.Path(__file__).parent / "hmd_cli_bartleby" / "skills"

if src_skills.exists() and src_skills.is_dir():
    # Remove existing skills dir (or symlink) and copy fresh
    if pkg_skills.is_symlink():
        pkg_skills.unlink()
    elif pkg_skills.exists():
        shutil.rmtree(pkg_skills)
    shutil.copytree(src_skills, pkg_skills)

setup(
    name="hmd-cli-bartleby",
    version=version,
    description="CLI for bartleby transform",
    author="Kate Walls",
    author_email="kate.walls@hmdlabs.io",
    license="BSL 1.1",
    packages=find_packages(),
    include_package_data=True,
    package_data={
        "hmd_cli_bartleby": [
            "agents/**/*",
            "agents/*",
            "skills/**/*",
            "skills/*",
        ],
    },
    install_requires=[],
)
