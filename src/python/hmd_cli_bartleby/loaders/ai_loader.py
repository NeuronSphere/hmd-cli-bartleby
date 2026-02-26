"""AI component loader for hmd-cli-bartleby.

Discovers and loads agents for use with the hmd ai command.
"""

import re
from pathlib import Path
from typing import Dict, List, Tuple

import yaml


def _get_skills_location() -> Path:
    """Get the path to the skills directory.

    Supports both installed package and development environments.
    """
    # First, try the package-relative location (for installed packages)
    package_skills = Path(__file__).parent.parent / "skills"
    if package_skills.exists():
        return package_skills

    # Fall back to src/skills/ for development
    # Path: loaders/ai_loader.py is at src/python/hmd_cli_bartleby/loaders/
    # We need to get to: src/skills/
    dev_skills = Path(__file__).parent.parent.parent.parent / "skills"
    if dev_skills.exists():
        return dev_skills

    # Return the package location even if it doesn't exist yet
    return package_skills


def _get_agents_location() -> Path:
    """Get the path to the agents directory.

    Supports both installed package and development environments.
    """
    # First, try the package-relative location (for installed packages)
    package_agents = Path(__file__).parent.parent / "agents"
    if package_agents.exists():
        return package_agents

    # Fall back to src/agents/ for development
    # Path: loaders/ai_loader.py is at src/python/hmd_cli_bartleby/loaders/
    # We need to get to: src/agents/
    # loaders -> hmd_cli_bartleby -> python -> src, then into agents
    dev_agents = Path(__file__).parent.parent.parent.parent / "agents"
    if dev_agents.exists():
        return dev_agents

    # Return the package location even if it doesn't exist yet
    return package_agents


DEFAULT_AGENTS_LOCATION = _get_agents_location()
DEFAULT_SKILLS_LOCATION = _get_skills_location()


class AILoader:
    """Loader for AI agent and skill files with YAML frontmatter metadata."""

    def __init__(
        self,
        agents_location: Path = DEFAULT_AGENTS_LOCATION,
        skills_location: Path = DEFAULT_SKILLS_LOCATION,
    ) -> None:
        self.agents_location = Path(agents_location)
        self.skills_location = Path(skills_location)
        # Keep default_location for backwards compatibility
        self.default_location = self.agents_location

    def _parse_frontmatter(self, content: str) -> Tuple[Dict, str]:
        """Parse YAML frontmatter from markdown content.

        Returns:
            Tuple of (metadata dict, remaining content)
        """
        # Match YAML frontmatter between --- markers
        pattern = r"^---\s*\n(.*?)\n---\s*\n(.*)$"
        match = re.match(pattern, content, re.DOTALL)

        if match:
            frontmatter_str = match.group(1)
            body = match.group(2)
            try:
                metadata = yaml.safe_load(frontmatter_str) or {}
            except yaml.YAMLError:
                metadata = {}
            return metadata, body
        else:
            return {}, content

    def list_commands(self) -> List[Dict]:
        """List all available commands.

        Returns:
            Empty list (no commands defined)
        """
        return []

    def list_skills(self) -> List[Dict]:
        """List all available skills with their metadata.

        Supports two formats:
        - File-based: skills/{name}.md
        - Directory-based: skills/{name}/SKILL.md

        Returns:
            List of skill metadata dictionaries
        """
        skills = []

        if not self.skills_location.exists():
            return skills

        # Find file-based skills (*.md directly in skills/)
        file_based = list(self.skills_location.glob("*.md"))

        # Find directory-based skills (*/SKILL.md)
        dir_based = list(self.skills_location.glob("*/SKILL.md"))

        skill_files = list(set(file_based + dir_based))  # Deduplicate

        for skill_file in sorted(skill_files):
            try:
                with open(skill_file, "r") as f:
                    content = f.read()
                metadata, _ = self._parse_frontmatter(content)

                # Add filename-based name if not in metadata
                if "name" not in metadata:
                    # For directory-based, use parent dir name
                    if skill_file.name == "SKILL.md":
                        metadata["name"] = skill_file.parent.name
                    else:
                        metadata["name"] = skill_file.stem

                # Add file path for reference (used by hmd ai install)
                metadata["_file"] = str(skill_file)

                skills.append(metadata)
            except Exception:
                # Skip files that can't be read
                continue

        return skills

    def list_agents(self) -> List[Dict]:
        """List all available agents with their metadata.

        Supports two formats:
        - File-based: agents/{name}.md or agents/{name}.AGENT.md
        - Directory-based: agents/{name}/AGENT.md

        Returns:
            List of agent metadata dictionaries
        """
        agents = []

        if not self.default_location.exists():
            return agents

        # Find file-based agents (*.md or *AGENT.md directly in agents/)
        file_based = list(self.default_location.glob("*.md"))
        file_based.extend(list(self.default_location.glob("*AGENT.md")))

        # Find directory-based agents (*/AGENT.md)
        dir_based = list(self.default_location.glob("*/AGENT.md"))

        agent_files = list(set(file_based + dir_based))  # Deduplicate

        for agent_file in sorted(agent_files):
            try:
                with open(agent_file, "r") as f:
                    content = f.read()
                metadata, _ = self._parse_frontmatter(content)

                # Add filename-based name if not in metadata
                if "name" not in metadata:
                    # For directory-based, use parent dir name
                    if agent_file.name == "AGENT.md":
                        metadata["name"] = agent_file.parent.name
                    else:
                        metadata["name"] = agent_file.stem

                # Add file path for reference (used by hmd ai install)
                metadata["_file"] = str(agent_file)

                agents.append(metadata)
            except Exception:
                # Skip files that can't be read
                continue

        return agents

    def load_agent(self, name: str) -> Tuple[Dict, str]:
        """Load an agent by name.

        Args:
            name: Agent name (filename without .md extension)

        Returns:
            Tuple of (metadata dict, full content including frontmatter)
        """
        agent_path = self.get_agent_path(name)

        with open(agent_path, "r") as f:
            content = f.read()

        metadata, _ = self._parse_frontmatter(content)
        if "name" not in metadata:
            metadata["name"] = name

        return metadata, content

    def get_agent_path(self, name: str) -> Path:
        """Get the path to an agent file.

        Supports two formats:
        - File-based: agents/{name}.md
        - Directory-based: agents/{name}/AGENT.md

        Args:
            name: Agent name (filename without .md extension or directory name)

        Returns:
            Path to the agent file

        Raises:
            FileNotFoundError: If agent doesn't exist
        """
        # Try file-based format first: {name}.md
        agent_path = self.default_location / f"{name}.md"
        if agent_path.exists():
            return agent_path

        # Try AGENT.md file format: {name}.AGENT.md
        agent_path = self.default_location / f"{name}.AGENT.md"
        if agent_path.exists():
            return agent_path

        # Try directory-based format: {name}/AGENT.md
        dir_agent_path = self.default_location / name / "AGENT.md"
        if dir_agent_path.exists():
            return dir_agent_path

        # Try finding by glob pattern (file-based)
        matches = list(self.default_location.glob(f"*{name}*.md"))
        if matches:
            return matches[0]

        # Try finding by glob pattern (directory-based)
        dir_matches = list(self.default_location.glob(f"*{name}*/AGENT.md"))
        if dir_matches:
            return dir_matches[0]

        raise FileNotFoundError(f"Agent '{name}' not found in {self.default_location}")

    def agent_exists(self, name: str) -> bool:
        """Check if an agent exists.

        Args:
            name: Agent name

        Returns:
            True if agent exists, False otherwise
        """
        try:
            self.get_agent_path(name)
            return True
        except FileNotFoundError:
            return False
