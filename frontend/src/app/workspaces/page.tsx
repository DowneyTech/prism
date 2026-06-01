"use client";

import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { workspaces, ApiError } from "@/lib/api";
import { useAuthStore } from "@/store/auth";

const schema = z.object({
  name: z.string().min(2, "Workspace name must be at least 2 characters").max(50),
});

type FormValues = z.infer<typeof schema>;

export default function WorkspacesPage() {
  const router = useRouter();
  const user = useAuthStore((s) => s.user);
  const clearAuth = useAuthStore((s) => s.clearAuth);
  const [serverError, setServerError] = useState("");

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({ resolver: zodResolver(schema) });

  async function onCreate(values: FormValues) {
    setServerError("");
    try {
      const ws = await workspaces.create(values.name);
      router.push(`/${ws.slug}/dashboard`);
    } catch (err) {
      if (err instanceof ApiError) {
        setServerError(err.message);
      } else {
        setServerError("Something went wrong. Please try again.");
      }
    }
  }

  function handleLogout() {
    clearAuth();
    router.push("/login");
  }

  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-muted/40 px-4">
      <div className="mb-6 text-center">
        <h1 className="text-3xl font-bold tracking-tight">Prism</h1>
        {user && (
          <p className="mt-1 text-sm text-muted-foreground">
            Signed in as <span className="font-medium text-foreground">{user.email}</span>
          </p>
        )}
      </div>

      <Card className="w-full max-w-sm">
        <CardHeader>
          <CardTitle>Create a workspace</CardTitle>
          <CardDescription>
            A workspace is your team&apos;s home in Prism. Give it your company or team name.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit(onCreate)} noValidate className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="name">Workspace name</Label>
              <Input
                id="name"
                type="text"
                placeholder="Acme Corp"
                autoFocus
                {...register("name")}
              />
              {errors.name && (
                <p className="text-xs text-destructive">{errors.name.message}</p>
              )}
            </div>

            {serverError && (
              <p className="text-sm text-destructive">{serverError}</p>
            )}

            <Button type="submit" className="w-full" disabled={isSubmitting}>
              {isSubmitting ? "Creating…" : "Create workspace"}
            </Button>
          </form>
        </CardContent>
      </Card>

      <button
        onClick={handleLogout}
        className="mt-6 text-sm text-muted-foreground underline underline-offset-4"
      >
        Sign out
      </button>
    </div>
  );
}
