package merry

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

// ContainerGroup represents a group configuration
type ContainerGroup struct {
	Description string   `yaml:"description"`
	Patterns    []string `yaml:"patterns"`
	Includes    []string `yaml:"includes"`
	Excludes    []string `yaml:"excludes"`
}

// ContainerGroups represents the entire groups configuration
type ContainerGroups struct {
	Groups map[string]ContainerGroup `yaml:"groups"`
}

// loadContainerGroups loads the container groups configuration
func (m *Merry) loadContainerGroups() (*ContainerGroups, error) {
	// Assuming the config file is in the same directory as the compose file
	composePath, err := defaultComposePath()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(filepath.Dir(composePath), "container-groups.yml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read container groups config: %v", err)
	}

	var groups ContainerGroups
	if err := yaml.Unmarshal(data, &groups); err != nil {
		return nil, fmt.Errorf("failed to parse container groups config: %v", err)
	}

	return &groups, nil
}

// getRunningContainers gets list of running container names
func (m *Merry) getRunningContainers() ([]string, error) {
	composePath, err := defaultComposePath()
	if err != nil {
		return nil, err
	}

	cmd := runDockerCompose(composePath, "ps", "--services", "--filter", "status=running")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get running containers: %v", err)
	}

	containers := strings.Split(strings.TrimSpace(string(output)), "\n")
	var result []string
	for _, container := range containers {
		if strings.TrimSpace(container) != "" {
			result = append(result, strings.TrimSpace(container))
		}
	}

	return result, nil
}

// matchContainersForGroup matches containers based on group configuration
func (m *Merry) matchContainersForGroup(group ContainerGroup, runningContainers []string) []string {
	matched := make(map[string]bool)

	for _, container := range runningContainers {
		for _, pattern := range group.Patterns {
			if m.matchPattern(strings.ToLower(container), strings.ToLower(pattern)) {
				matched[container] = true
			}
		}
	}

	// Add explicit includes
	for _, include := range group.Includes {
		for _, container := range runningContainers {
			if container == include {
				matched[container] = true
			}
		}
	}

	// Remove excludes
	for _, exclude := range group.Excludes {
		delete(matched, exclude)
	}

	result := make([]string, 0, len(matched))
	for container := range matched {
		result = append(result, container)
	}

	return result
}

func (m *Merry) matchPattern(text, pattern string) bool {
	if pattern == "*" {
		return true
	}

	if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
		// *text* - contains
		substr := pattern[1 : len(pattern)-1]
		return strings.Contains(text, substr)
	}

	if strings.HasPrefix(pattern, "*") {
		// *text - ends with
		suffix := pattern[1:]
		return strings.HasSuffix(text, suffix)
	}

	if strings.HasSuffix(pattern, "*") {
		// text* - starts with
		prefix := pattern[:len(pattern)-1]
		return strings.HasPrefix(text, prefix)
	}

	// Exact match
	return text == pattern
}


func (m *Merry) getAllContainers() ([]string, error) {
	composePath, err := defaultComposePath()
	if err != nil {
		return nil, err
	}

	cmd := runDockerCompose(composePath, "ps", "--services")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get containers: %v", err)
	}

	containers := strings.Split(strings.TrimSpace(string(output)), "\n")
	var result []string
	for _, container := range containers {
		if strings.TrimSpace(container) != "" {
			result = append(result, strings.TrimSpace(container))
		}
	}

	return result, nil
}

func (m *Merry) ListGroups() error {
	// Load container groups configuration
	groups, err := m.loadContainerGroups()
	if err != nil {
		return fmt.Errorf("failed to load container groups: %v", err)
	}

	// Get all containers (both running and stopped)
	allContainers, err := m.getAllContainers()
	if err != nil {
		return fmt.Errorf("failed to get containers: %v", err)
	}

	// Get running containers to show status
	runningContainers, err := m.getRunningContainers()
	if err != nil {
		return fmt.Errorf("failed to get running containers: %v", err)
	}

	// Create a map for quick lookup of running containers
	runningMap := make(map[string]bool)
	for _, container := range runningContainers {
		runningMap[container] = true
	}

	fmt.Println("Container Groups:")
	fmt.Println("================")

	for groupName, group := range groups.Groups {
		fmt.Printf("\n%s:\n", strings.ToUpper(groupName))
		fmt.Printf("  Description: %s\n", group.Description)
		
		// Find matching containers
		matched := m.matchContainersForGroup(group, allContainers)
		
		if len(matched) == 0 {
			fmt.Printf("  Containers: None found\n")
		} else {
			fmt.Printf("  Containers (%d):\n", len(matched))
			for _, container := range matched {
				status := "stopped"
				if runningMap[container] {
					status = "running"
				}
				fmt.Printf("    - %s [%s]\n", container, status)
			}
		}
	}
	return nil
}