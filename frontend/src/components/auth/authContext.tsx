import { createContext, useContext } from "react";

export interface AuthUser {
  username: string;
  token: string;
}

export interface AuthContextType {
  user: AuthUser | null;
  signin: (username: string, password: string, callback: VoidFunction) => void;
  signout: (callback: VoidFunction) => void;
}

export const AuthContext = createContext<AuthContextType>(null!);

export const useAuth = () => {
  return useContext(AuthContext);
};
