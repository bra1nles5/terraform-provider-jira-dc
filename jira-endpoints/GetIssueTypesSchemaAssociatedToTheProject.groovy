import com.atlassian.jira.component.ComponentAccessor
import com.atlassian.jira.project.Project
import com.atlassian.jira.issue.fields.config.manager.IssueTypeSchemeManager
import com.atlassian.jira.issue.fields.config.FieldConfigScheme
import com.atlassian.jira.issue.issuetype.IssueType  // for the defaultIssueType type
import com.onresolve.scriptrunner.runner.rest.common.CustomEndpointDelegate
import groovy.json.JsonBuilder
import groovy.transform.BaseScript
import javax.ws.rs.core.MultivaluedMap
import javax.ws.rs.core.Response

@BaseScript CustomEndpointDelegate delegate

getIssueTypeScheme(
    httpMethod: "GET",
    groups: ["jira-administrators"]
) { MultivaluedMap queryParams, String body ->

    def projectKey = queryParams.getFirst("projectKey") ?: queryParams.getFirst("key")

    if (!projectKey) {
        return Response.status(400).entity(new JsonBuilder([error: "Required query param: projectKey"]).toString()).build()
    }

    def projectManager = ComponentAccessor.projectManager
    def schemeManager = ComponentAccessor.getIssueTypeSchemeManager()

    Project project = projectManager.getProjectObjByKeyIgnoreCase(projectKey)
    if (!project) {
        return Response.status(404).entity(new JsonBuilder([error: "Project '${projectKey}' not found"]).toString()).build()
    }

    FieldConfigScheme scheme = schemeManager.getConfigScheme(project)

    if (!scheme) {
        return Response.status(404).entity(new JsonBuilder([error: "No Issue Type Scheme associated with project"]).toString()).build()
    }

    // Correct way to obtain the default issue type
    IssueType defaultIssueType = schemeManager.getDefaultIssueType(project)

    // Issue types available in the project. getAssociatedIssueTypes() is the field
    // context, not the scheme contents, and for a global scheme it is [null].
    def issueTypes = schemeManager.getIssueTypesForProject(project)*.name ?: []

    def response = [
        projectKey          : projectKey,
        schemeId            : scheme.id,
        schemeName          : scheme.name,
        schemeDescription   : scheme.description ?: null,
        defaultIssueType    : defaultIssueType?.name ?: null,  // default issue type name, or null
        issueTypes          : issueTypes
    ]

    return Response.ok(new JsonBuilder(response).toString()).build()
}