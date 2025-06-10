package merry

import (
	"fmt"
	"os"
	"strings"
)

func (m *Merry) Disable(groupNames ...string) error {
	if len(groupNames) == 0 {
		return fmt.Errorf("no groups specified")
	}

	groups, err := m.loadContainerGroups()
	if err != nil {
		return fmt.Errorf("failed to load container groups: %v", err)
	}

	runningContainers, err := m.getRunningContainers()
	if err != nil {
		return fmt.Errorf("failed to get running containers: %v", err)
	}

	containersToStop := make(map[string]bool)
	for _, groupName := range groupNames {
		group, exists := groups.Groups[groupName]
		if !exists {
			fmt.Printf("Warning: Group '%s' not found\n", groupName)
			continue
		}

		matched := m.matchContainersForGroup(group, runningContainers)
		for _, container := range matched {
			containersToStop[container] = true
		}
	}

	if len(containersToStop) == 0 {
		fmt.Println("No containers found matching the specified groups")
		return nil
	}

	containerList := make([]string, 0, len(containersToStop))
	for container := range containersToStop {
		containerList = append(containerList, container)
	}

	fmt.Printf("Stopping containers: %s\n", strings.Join(containerList, ", "))

	composePath, err := defaultComposePath()
	if err != nil {
		return err
	}

	args := []string{"stop"}
	args = append(args, containerList...)
	
	bashCmd := runDockerCompose(composePath, args...)
	bashCmd.Stdout = os.Stdout
	bashCmd.Stderr = os.Stderr

	if err := bashCmd.Run(); err != nil {
		return fmt.Errorf("failed to stop containers: %v", err)
	}

	fmt.Printf("Successfully stopped %d containers from groups: %s\n", 
		len(containerList), strings.Join(groupNames, ", "))

	return nil
}


func (m *Merry) Enable(groupNames ...string) error {
	if len(groupNames) == 0 {
		return fmt.Errorf("no groups specified")
	}

	groups, err := m.loadContainerGroups()
	if err != nil {
		return fmt.Errorf("failed to load container groups: %v", err)
	}

	allContainers, err := m.getAllContainers()
	if err != nil {
		return fmt.Errorf("failed to get containers: %v", err)
	}

	containersToStart := make(map[string]bool)
	for _, groupName := range groupNames {
		group, exists := groups.Groups[groupName]
		if !exists {
			fmt.Printf("Warning: Group '%s' not found\n", groupName)
			continue
		}

		matched := m.matchContainersForGroup(group, allContainers)
		for _, container := range matched {
			containersToStart[container] = true
		}
	}

	if len(containersToStart) == 0 {
		fmt.Println("No containers found matching the specified groups")
		return nil
	}

	containerList := make([]string, 0, len(containersToStart))
	for container := range containersToStart {
		containerList = append(containerList, container)
	}

	fmt.Printf("Starting containers: %s\n", strings.Join(containerList, ", "))

	composePath, err := defaultComposePath()
	if err != nil {
		return err
	}

	args := []string{"start"}
	args = append(args, containerList...)
	
	bashCmd := runDockerCompose(composePath, args...)
	bashCmd.Stdout = os.Stdout
	bashCmd.Stderr = os.Stderr

	if err := bashCmd.Run(); err != nil {
		return fmt.Errorf("failed to start containers: %v", err)
	}

	fmt.Printf("Successfully started %d containers from groups: %s\n", 
		len(containerList), strings.Join(groupNames, ", "))

	return nil
}