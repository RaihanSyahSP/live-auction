import { z } from "zod";

export const loginSchema = z.object({
  email: z.string().email("Email tidak valid"),
  password: z.string().min(1, "Kata sandi tidak boleh kosong"),
});

export const registerSchema = z
  .object({
    email: z.string().email("Email tidak valid"),
    password: z
      .string()
      .min(8, "Kata sandi harus minimal 8 karakter")
      .max(72, "Kata sandi maksimal 72 karakter"),
    confirmPassword: z.string().min(1, "Ulangi kata sandi"),
    displayName: z
      .string()
      .min(3, "Nama tampilan minimal 3 karakter")
      .max(50, "Nama tampilan maksimal 50 karakter"),
  })
  .refine((data) => data.password === data.confirmPassword, {
    message: "Kata sandi dan konfirmasi kata sandi tidak cocok",
    path: ["confirmPassword"],
  });

export type RegisterSchema = z.infer<typeof registerSchema>;
export type LoginSchema = z.infer<typeof loginSchema>;
