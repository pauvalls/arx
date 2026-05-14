package application

import "strings"

// explanations is the built-in library of architectural explanations.
// These are the "teaching soul" of Arx — clear, actionable, and educational.
var explanations = map[string]string{
	// Layer-specific patterns
	"domain-imports-infrastructure": "The domain layer is the heart of your business logic. It should NEVER depend on infrastructure concerns like databases, HTTP frameworks, or external APIs. Dependencies should flow inward: infrastructure depends on domain, not the reverse.",

	"domain-imports-application": "The domain layer is the heart of your business logic. It should remain pure and independent of application orchestration concerns. Application services coordinate domain operations, but domain entities must not know about them.",

	"application-imports-infrastructure": "The application layer exists to orchestrate domain operations. It should not directly depend on infrastructure implementations. Use interfaces (ports) defined in the application layer and implemented in the infrastructure layer.",

	"infrastructure-imports-domain": "Infrastructure implements adapters and should not be imported directly by other infrastructure modules. If infrastructure needs to reference domain concepts, it should do so through well-defined interfaces (ports) rather than concrete implementations.",

	"presentation-imports-infrastructure": "The presentation layer (HTTP handlers, CLI controllers, UI) should not bypass the application layer to reach infrastructure directly. This creates tight coupling and makes your system hard to test and evolve.",

	"presentation-imports-domain": "The presentation layer should delegate to application services rather than directly manipulating domain entities. This keeps your domain logic centralized and your presentation layer thin.",

	// Circular dependency patterns
	"domain-circular": "Circular dependencies create maintenance nightmares and prevent independent testing. If two domain concepts depend on each other, consider extracting a shared abstraction or rethinking your aggregate boundaries.",

	"application-circular": "Circular dependencies in the application layer suggest services are doing too much or responsibilities are poorly separated. Consider splitting services by use case or introducing an intermediate abstraction.",

	"infrastructure-circular": "Circular dependencies in infrastructure often mean adapters are tightly coupled. Use dependency inversion: define interfaces in the application layer and inject implementations.",

	"layer-circular": "Circular dependencies between layers violate the fundamental principle of layered architecture: dependencies should flow in one direction. Review your layer boundaries and apply the Dependency Inversion Principle.",

	// General patterns
	"default": "This dependency violates an architectural rule. Review your layer boundaries and ensure dependencies flow in the correct direction according to your chosen architecture.",
}

// fixGuidance provides actionable steps to resolve violations.
var fixGuidance = map[string][]string{
	"domain-imports-infrastructure": {
		"Move the infrastructure concern behind an interface (port) defined in the domain or application layer",
		"Use dependency injection to provide the implementation at runtime",
		"Consider using the Repository pattern to abstract data access",
	},

	"domain-imports-application": {
		"Move the shared logic into the domain layer or extract a domain service",
		"Use domain events instead of direct application service calls",
		"Ensure domain entities remain independent of orchestration concerns",
	},

	"application-imports-infrastructure": {
		"Define an interface (port) in the application layer for the infrastructure concern",
		"Move the concrete implementation to the infrastructure layer",
		"Inject the interface implementation via constructor or dependency injection container",
	},

	"infrastructure-imports-domain": {
		"Ensure infrastructure only references domain through well-defined interfaces",
		"Avoid direct instantiation of domain objects in infrastructure code",
		"Use factory methods or builders defined in the application layer",
	},

	"presentation-imports-infrastructure": {
		"Create an application service method that encapsulates the infrastructure interaction",
		"Have the presentation layer call the application service instead",
		"This keeps your presentation layer thin and testable",
	},

	"presentation-imports-domain": {
		"Create an application service that coordinates the domain operation",
		"Pass DTOs between presentation and application layers",
		"Keep domain logic encapsulated behind application services",
	},

	"domain-circular": {
		"Extract the shared logic into a third concept that both can depend on",
		"Consider whether one aggregate should reference the other by ID instead of object",
		"Review your bounded context boundaries — the concepts may belong in separate contexts",
	},

	"application-circular": {
		"Split the services by use case or responsibility",
		"Introduce a mediator or command bus to decouple services",
		"Extract shared logic into a domain service or utility",
	},

	"infrastructure-circular": {
		"Define interfaces in the application layer and inject implementations",
		"Use a dependency injection container to wire dependencies",
		"Consider the Adapter pattern to decouple concrete implementations",
	},

	"layer-circular": {
		"Draw your dependency diagram and identify the cycle",
		"Apply the Dependency Inversion Principle: depend on abstractions, not concretions",
		"Consider introducing an intermediate layer or shared kernel",
	},

	"default": {
		"Review the rule definition in arx.yaml to understand the intent",
		"Trace the dependency path to find where the rule is violated",
		"Refactor to ensure dependencies flow in the correct direction",
	},
}

// GetExplanation returns a human-readable explanation for a given rule ID.
// It uses pattern matching to find the most specific explanation available.
func GetExplanation(ruleID string) string {
	// Try exact match first
	if explanation, ok := explanations[ruleID]; ok {
		return explanation
	}

	// Try prefix/suffix pattern matching
	for pattern, explanation := range explanations {
		if pattern == "default" {
			continue
		}

		// Check for prefix match (e.g., "domain-*" matches "domain-imports-infrastructure")
		if strings.HasSuffix(pattern, "-*") {
			prefix := strings.TrimSuffix(pattern, "-*")
			if strings.HasPrefix(ruleID, prefix) {
				return explanation
			}
		}

		// Check for suffix match (e.g., "*-circular" matches "domain-circular")
		if strings.HasPrefix(pattern, "*-") {
			suffix := strings.TrimPrefix(pattern, "*-")
			if strings.HasSuffix(ruleID, suffix) {
				return explanation
			}
		}
	}

	return explanations["default"]
}

// GetFixGuidance returns actionable steps to resolve a violation for a given rule ID.
func GetFixGuidance(ruleID string) []string {
	// Try exact match first
	if guidance, ok := fixGuidance[ruleID]; ok {
		return guidance
	}

	// Try prefix/suffix pattern matching
	for pattern, guidance := range fixGuidance {
		if pattern == "default" {
			continue
		}

		// Check for prefix match
		if strings.HasSuffix(pattern, "-*") {
			prefix := strings.TrimSuffix(pattern, "-*")
			if strings.HasPrefix(ruleID, prefix) {
				return guidance
			}
		}

		// Check for suffix match
		if strings.HasPrefix(pattern, "*-") {
			suffix := strings.TrimPrefix(pattern, "*-")
			if strings.HasSuffix(ruleID, suffix) {
				return guidance
			}
		}
	}

	return fixGuidance["default"]
}
