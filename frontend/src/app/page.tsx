import { cookies } from "next/headers";
import { redirect } from "next/navigation";

export default function RootPage() {
  const token = cookies().get("token");
  redirect(token ? "/workspaces" : "/login");
}
