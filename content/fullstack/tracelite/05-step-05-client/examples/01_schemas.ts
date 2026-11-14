// LEARN(fullstack.tracelite.step-05-client): generated types say what the contract is; zod schemas tied to them check it at run time.
import { z } from "zod";

// --- src/api/schema.d.ts: openapi-typescript output (abridged), types only.
interface components {
  schemas: {
    Requirement: {
      /** Format: uuid */
      id: string;
      /** Format: uuid */
      projectId: string;
      parentId: string | null;
      code: string;
      /** @enum {string} */
      status: "draft" | "approved" | "obsolete";
      /** Format: date-time */
      createdAt: string;
    };
  };
}

// --- src/api/schemas.ts: the run-time check, tied to the generated type.
const ProjectId = z.uuid().brand<"ProjectId">();
type ProjectId = z.infer<typeof ProjectId>;
const RequirementId = z.uuid().brand<"RequirementId">();
const Requirement = z.object({
  id: RequirementId,
  projectId: ProjectId,
  parentId: RequirementId.nullable(),
  code: z.string(),
  status: z.enum(["draft", "approved", "obsolete"]),
  createdAt: z.iso.datetime(), // UTC with Z only, as the Go side sends it
}) satisfies z.ZodType<components["schemas"]["Requirement"]>;

const requirementsOf = (projectId: ProjectId) =>
  `/api/projects/${projectId}/requirements`;

// What GET /api/requirements/{requirementId} answers.
const body = {
  id: "01937b6e-0000-7000-8000-000000000002",
  projectId: "01937b6e-0000-7000-8000-00000000000a",
  parentId: null,
  code: "AUTH-LOGIN",
  status: "draft",
  createdAt: "2026-11-13T06:30:00.123456Z",
};
const requirement = Requirement.parse(body); // branded ids from here on
console.log(`${requirement.code}: ${requirementsOf(requirement.projectId)}`);
// requirementsOf(requirement.id);
//   TS2345: Argument of type 'string & $brand<"RequirementId">' is not
//   assignable to parameter of type 'string & $brand<"ProjectId">'.

// A Go handler that forgot .UTC(): the same instant, but not the contract.
const local = { ...body, createdAt: "2026-11-13T09:30:00.123456+03:00" };
const result = Requirement.safeParse(local);
if (!result.success) console.log(z.prettifyError(result.error));
