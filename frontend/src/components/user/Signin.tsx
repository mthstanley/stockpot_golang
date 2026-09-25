import { useLocation, useNavigate } from "react-router";
import { useAuth } from "../auth/authContext";
import { useForm } from "react-hook-form";

type FormValues = {
  username: string;
  password: string;
};

const SigninPage = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const auth = useAuth();
  const { register, handleSubmit } = useForm<FormValues>();

  const from = location.state?.from?.pathname || "/";

  const onSubmit = handleSubmit((credentials: FormValues) => {
    auth.signin(credentials.username, credentials.password, () => {
      // Send them back to the page they tried to visit when they were
      // redirected to the login page. Use { replace: true } so we don't create
      // another entry in the history stack for the login page.  This means that
      // when they get to the protected page and click the back button, they
      // won't end up back on the login page, which is also really nice for the
      // user experience.
      navigate(from, { replace: true });
    });
  });

  return (
    <div className="auth">
      <header className="auth-header page-title">
        <hgroup>
          <h1>Sign-In</h1>
          <p>You must log in to view the page at {from}</p>
        </hgroup>
      </header>

      <div className="auth-body">
        <form onSubmit={onSubmit}>
          <p>
            <label htmlFor="username">Username</label>
            <input
              id="username"
              {...register("username", { required: true })}
            />
          </p>
          <p>
            <label htmlFor="password">Password</label>
            <input
              type="password"
              id="password"
              {...register("password", { required: true })}
            />
          </p>
          <p>
            <input type="submit" value="Sign-in" />
          </p>
        </form>
      </div>
    </div>
  );
};

export default SigninPage;
