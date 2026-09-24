import { useNavigate } from "react-router";
import { useAuth } from "../auth/authContext";
import { useEffect } from "react";

const Signout = () => {
  const auth = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    auth.signout(() => navigate("/"));
  }, [auth, navigate]);

  return null;
};

export default Signout;
