import { createContext } from "react-router";
import type { User } from "~/features/auth/types";

export const userContext = createContext<User | null>(null);
