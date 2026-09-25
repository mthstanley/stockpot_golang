import { useAuth } from "../auth/authContext";

const ProfilePage = () => {
  const auth = useAuth();

  if (!auth.user) {
    return <p>You are not logged in.</p>;
  }

  return (
    <>
      <header className="page-title">
        <hgroup>
          <h1>User Profile</h1>
          <p>Welcome {auth.user.username}!</p>
        </hgroup>
      </header>
    </>
  );
};

export default ProfilePage;
