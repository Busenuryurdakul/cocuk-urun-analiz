"use client";

import Link from "next/link";
import { FormEvent, useState } from "react";
import { AsyncView } from "@/components/async-view";
import { graphqlRequest } from "@/lib/graphql";

export default function RegisterPage() {
  const [state, setState] = useState<"idle" | "loading" | "success" | "error">("idle");
  const [message, setMessage] = useState("");

  async function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setState("loading");
    const form = new FormData(e.currentTarget);
    try {
      const data = await graphqlRequest<{ register: { message: string } }>(
        `mutation Register($input: RegisterInput!) {
          register(input: $input) { message }
        }`,
        {
          input: {
            email: String(form.get("email")),
            password: String(form.get("password")),
          },
        },
      );
      setMessage(data.register.message);
      setState("success");
    } catch (err) {
      setMessage(err instanceof Error ? err.message : "Kayıt başarısız");
      setState("error");
    }
  }

  return (
    <main className="mx-auto flex min-h-screen max-w-md flex-col gap-6 px-6 py-16">
      <header>
        <p className="text-sm font-medium uppercase tracking-wide text-miyuna-600">Miyuna</p>
        <h1 className="text-2xl font-semibold text-slate-900">Hesap oluştur</h1>
      </header>

      {state === "loading" && <AsyncView state="loading" />}
      {state === "error" && (
        <AsyncView state="error" error={<p className="text-sm text-red-800">{message}</p>} />
      )}
      {state === "success" ? (
        <div className="rounded-lg border border-emerald-200 bg-emerald-50 p-6 text-sm text-emerald-900">
          <p>{message}</p>
          <Link className="mt-4 inline-block text-miyuna-600 underline" href="/auth/login">
            Giriş sayfasına git
          </Link>
        </div>
      ) : state === "idle" ? (
        <form onSubmit={onSubmit} className="space-y-4 rounded-xl border border-slate-200 bg-white p-6">
          <label className="block text-sm">
            E-posta
            <input
              name="email"
              type="email"
              required
              className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2"
            />
          </label>
          <label className="block text-sm">
            Şifre (min. 8 karakter)
            <input
              name="password"
              type="password"
              minLength={8}
              required
              className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2"
            />
          </label>
          <button
            type="submit"
            className="w-full rounded-md bg-miyuna-600 px-4 py-2 text-sm font-medium text-white hover:bg-sky-700"
          >
            Kayıt ol
          </button>
        </form>
      ) : null}

      <Link href="/auth/login" className="text-sm text-slate-600 underline">
        Zaten hesabın var mı? Giriş yap
      </Link>
    </main>
  );
}
