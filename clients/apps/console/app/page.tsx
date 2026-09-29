import { Button } from "@leamout/ui/components/button";

export default function Home() {
  return (
    <main className="flex min-h-screen items-center justify-center">
      <div className="space-y-4 text-center">
        <h1 className="font-heading text-4xl font-semibold">Hello, Leamout</h1>

        <p className="text-muted-foreground">
          Communications infrastructure for developers.
        </p>

        <Button>Get started</Button>
      </div>
    </main>
  );
}
